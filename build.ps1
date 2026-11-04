param([string]$GoExe='go', [string]$OutputDirectory='')
$ErrorActionPreference='Stop'
$root=$PSScriptRoot
$version=[IO.File]::ReadAllText((Join-Path $root 'VERSION')).Trim()
$source=Join-Path $root 'src'
if (!$OutputDirectory) { $OutputDirectory=Join-Path $root ('dist\OKIGAMI-'+$version) }
$output=[IO.Path]::GetFullPath($OutputDirectory)
$package=Join-Path $output 'Kindle-USB'
$app=Join-Path $package 'kindle-board'
$state=Join-Path $app 'state'
if ((Test-Path -LiteralPath $state) -and @(Get-ChildItem -LiteralPath $state -File -Recurse -Force).Count -gt 0) { throw 'Output contains runtime data. Choose a clean output directory.' }
New-Item -ItemType Directory -Path (Join-Path $package 'documents'),(Join-Path $app 'bin'),$state,(Join-Path $app 'licenses') -Force | Out-Null
& $GoExe -C $source mod download
if ($LASTEXITCODE -ne 0) { throw 'Module download failed' }
& $GoExe -C $source test ./...
if ($LASTEXITCODE -ne 0) { throw 'Go tests failed' }
& $GoExe -C $source vet ./...
if ($LASTEXITCODE -ne 0) { throw 'Go vet failed' }
$oldCgo=$env:CGO_ENABLED; $oldOS=$env:GOOS; $oldArch=$env:GOARCH; $oldArm=$env:GOARM
try {
    $env:CGO_ENABLED='0'; $env:GOOS='windows'; $env:GOARCH='amd64'; Remove-Item Env:GOARM -ErrorAction SilentlyContinue
    & $GoExe -C $source build -buildvcs=false -trimpath -ldflags '-s -w' -o (Join-Path $output 'okigami-preview.exe') .
    if ($LASTEXITCODE -ne 0) { throw 'Preview build failed' }
    $env:GOOS='linux'; $env:GOARCH='arm'; $env:GOARM='7'
    & $GoExe -C $source build -buildvcs=false -trimpath -ldflags '-s -w' -o (Join-Path $app 'bin\kindle-board') .
    if ($LASTEXITCODE -ne 0) { throw 'Kindle build failed' }
} finally {
    $env:CGO_ENABLED=$oldCgo; $env:GOOS=$oldOS; $env:GOARCH=$oldArch; $env:GOARM=$oldArm
}
$upstream=Join-Path $root 'vendor\custom-screensaver'
$ss=Join-Path $app 'ss'
New-Item -ItemType Directory -Path $ss -Force | Out-Null
foreach ($name in @('bin','licenses','LICENSE','THIRD_PARTY_NOTICES.md','build-metadata.txt')) {
    Copy-Item -LiteralPath (Join-Path $upstream $name) -Destination $ss -Recurse -Force
}
foreach ($name in @('custom_ss_daemon.sh','blanket_renderers.sh','toggle.sh')) {
    $text=[IO.File]::ReadAllText((Join-Path $upstream $name)).Replace('/mnt/us/screensavers','/mnt/us/kindle-board/state/sleep').Replace('/tmp/custom_ss','/tmp/kindle_board_ss')
    [IO.File]::WriteAllText((Join-Path $ss $name),$text.Replace("`r`n","`n"),[Text.UTF8Encoding]::new($false))
}
foreach ($name in @('run.sh','display.sh','process.sh')) {
    $text=[IO.File]::ReadAllText((Join-Path $root ('device\'+$name)))
    [IO.File]::WriteAllText((Join-Path $app $name),$text.Replace("`r`n","`n"),[Text.UTF8Encoding]::new($false))
}
foreach ($name in @('OKIGAMI.sh','OKIGAMI Stop.sh')) {
    $text=[IO.File]::ReadAllText((Join-Path $root ('device\'+$name)))
    [IO.File]::WriteAllText((Join-Path $package ('documents\'+$name)),$text.Replace("`r`n","`n"),[Text.UTF8Encoding]::new($false))
}
$goRoot=(& $GoExe env GOROOT).Trim()
$goPath=(& $GoExe env GOPATH).Trim().Split([IO.Path]::PathSeparator)[0]
Copy-Item -LiteralPath (Join-Path $goRoot 'LICENSE') -Destination (Join-Path $app 'licenses\Go-LICENSE') -Force
Copy-Item -LiteralPath (Join-Path $goPath 'pkg\mod\github.com\skip2\go-qrcode@v0.0.0-20200617195104-da1b6568686e\LICENSE') -Destination (Join-Path $app 'licenses\go-qrcode-LICENSE') -Force
Copy-Item -LiteralPath (Join-Path $goPath 'pkg\mod\github.com\jezek\xgb@v1.1.1\LICENSE') -Destination (Join-Path $app 'licenses\xgb-LICENSE') -Force
foreach ($family in @('bizudgothic','bizudmincho','zenmarugothic')) {
    Copy-Item -LiteralPath (Join-Path $source ('web\fonts\'+$family+'-OFL.txt')) -Destination (Join-Path $app ('licenses\'+$family+'-OFL.txt')) -Force
}
foreach ($name in @('README.md','LICENSE','THIRD_PARTY_NOTICES.md','SOURCES.md')) { Copy-Item -LiteralPath (Join-Path $root $name) -Destination $output -Force }
Copy-Item -LiteralPath (Join-Path $root 'docs') -Destination $output -Recurse -Force
$manifest=@(Get-ChildItem -LiteralPath $package -File -Recurse | Sort-Object FullName | ForEach-Object {
    [PSCustomObject]@{Path=$_.FullName.Substring($package.Length+1).Replace('\','/');Bytes=$_.Length;SHA256=(Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()}
})
$manifest | ConvertTo-Json -Depth 3 | Set-Content -LiteralPath (Join-Path $output 'manifest.json') -Encoding utf8
Write-Output ('BUILD_OK: '+$output)
