# Third-party notices

OKIGAMI's own Go application, browser editor, and original support code are licensed under [MIT-0](LICENSE). This does not replace the licenses of the following components.

| Component | Version / source | License / notice |
|---|---|---|
| Go runtime and standard library | Go 1.27.3 for this release | BSD; `Kindle-USB/kindle-board/licenses/Go-LICENSE` |
| go-qrcode | `github.com/skip2/go-qrcode`, `da1b6568686e` | MIT; `licenses/go-qrcode-LICENSE` in the app package |
| XGB | `github.com/jezek/xgb`, v1.1.1 | See `licenses/xgb-LICENSE` in the app package |
| BIZ UDGothic, regular and bold | [Google Fonts](https://github.com/google/fonts/tree/main/ofl/bizudgothic) | SIL OFL 1.1; `src/web/fonts/bizudgothic-OFL.txt` |
| BIZ UDMincho, regular and bold | [Google Fonts](https://github.com/google/fonts/tree/main/ofl/bizudmincho) | SIL OFL 1.1; `src/web/fonts/bizudmincho-OFL.txt` |
| Zen Maru Gothic, regular and bold | [Google Fonts](https://github.com/google/fonts/tree/main/ofl/zenmarugothic) | SIL OFL 1.1; `src/web/fonts/zenmarugothic-OFL.txt` |
| Kindle Custom Screensaver | [André Cheng's project](https://github.com/chengandre/kindle-custom-screensaver), v0.6.0 | MIT for its original code; `vendor/custom-screensaver/LICENSE` |
| FBInk and `screensaver_draw` linked to it | [NiLuJe/FBInk](https://github.com/NiLuJe/FBInk), pinned revision below | GPL-3.0-or-later; `vendor/custom-screensaver/licenses/FBInk/LICENSE` and `CREDITS` |

Fonts are distributed as their original, unmodified TrueType files. Their source URLs, sizes, and Git blob hashes are recorded alongside them under `src/web/fonts/*-source.json`. Output images do not contain embedded font software.

The release package preserves Kindle Custom Screensaver's `LICENSE`, `THIRD_PARTY_NOTICES.md`, FBInk `LICENSE`/`CREDITS`, and `build-metadata.txt`. The helper shell scripts are adapted only to use OKIGAMI's internal state directory and temporary names. The native third-party executables are unmodified from the upstream release. OKIGAMI calls them as separate programs; its Go executable does not link FBInk.

## Corresponding source for bundled native helpers

[SOURCES.md](SOURCES.md) identifies the exact revisions and build inputs. The fixed source snapshots, the FBInk submodule source used by the Kindle build, and upstream build scripts are provided as **`OKIGAMI-0.6.1-third-party-sources.zip`** in the [same release](https://github.com/UroCo-lab/OKIGAMI/releases/tag/v0.6.1).

Do not treat the repository's MIT-0 license as permission to remove third-party license requirements from redistributed bundles.
