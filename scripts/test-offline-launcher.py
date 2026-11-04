"""Run the actual launcher with offline network/device commands simulated."""
from pathlib import Path
import os
import subprocess
import sys
import tempfile

root = Path(__file__).resolve().parent.parent
shell = sys.argv[1] if len(sys.argv) > 1 else 'sh'

def unix(path):
    value = str(path.resolve()).replace('\\', '/')
    return '/' + value[0].lower() + value[2:] if len(value) > 1 and value[1] == ':' else value

def write(path, text):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding='utf-8', newline='\n')
    path.chmod(0o755)

for firewall in (True, False):
    with tempfile.TemporaryDirectory(prefix='okigami-offline-') as folder:
        fixture = Path(folder)
        base = fixture / 'app'
        mock = fixture / 'commands'
        (base / 'state').mkdir(parents=True)
        (fixture / 'tmp').mkdir()
        source = (root / 'device/run.sh').read_text(encoding='utf-8')
        source = source.replace('BASE=/mnt/us/kindle-board', "BASE='" + unix(base) + "'")
        source = source.replace('/tmp/kindle_board_', unix(fixture / 'tmp') + '/kindle_board_')
        write(base / 'run.sh', source)
        write(base / 'process.sh', '#!/bin/sh\nexit 0\n')
        write(base / 'ss/toggle.sh', '#!/bin/sh\nexit 0\n')
        write(base / 'ss/blanket_renderers.sh', 'capture_renderer_state() { return 0; }\ndisable_original_renderers() { return 0; }\nrestore_original_renderers() { return 0; }\n')
        write(base / 'ss/bin/fbink_hf', '#!/bin/sh\necho UNEXPECTED_WIFI_ERROR_SCREEN\nexit 9\n')
        write(base / 'bin/kindle-board', '#!/bin/sh\nprintf "%s\\n" "$@" > "'+unix(fixture / 'receiver-args.txt')+'"\nexit 0\n')
        write(mock / 'ifconfig', '#!/bin/sh\nexit 1\n')
        write(mock / 'iptables', '#!/bin/sh\nexit '+('0' if firewall else '1')+'\n')
        write(mock / 'lipc-get-prop', '#!/bin/sh\necho screenSaver\n')
        write(mock / 'lipc-set-prop', '#!/bin/sh\nexit 0\n')
        write(mock / 'xrefresh', '#!/bin/sh\nexit 0\n')
        environment = os.environ.copy()
        environment['PATH'] = str(mock) + os.pathsep + str(Path(shell).resolve().parent) + os.pathsep + environment['PATH']
        result = subprocess.run([shell, str(base / 'run.sh').replace('\\','/'), 'worker'], env=environment, text=True, capture_output=True, timeout=15)
        assert result.returncode == 0, result.stdout + result.stderr
        assert (fixture / 'receiver-args.txt').exists(), 'Offline launcher never reached the app'
        assert 'UNEXPECTED_WIFI_ERROR_SCREEN' not in result.stdout
        log = (base / 'state/runtime.log').read_text()
        assert 'Starting offline with saved content' in log
        if not firewall:
            assert 'local display remains available' in log
        print('PASS: no Wi-Fi, firewall '+('available' if firewall else 'unavailable')+', app launched')
