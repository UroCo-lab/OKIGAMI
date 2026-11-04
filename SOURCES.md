# Native helper source and build inputs

OKIGAMI 0.6.0 bundles the unmodified `fbink_hf`, `screensaver_draw`, and `screensaver_shield` executables from:

- Release: <https://github.com/chengandre/kindle-custom-screensaver/releases/tag/v0.6.0>
- Original asset: `custom-screensaver-0.6.0-kindlehf.zip`
- Project commit: `c73cf75c47c0cc55a31464ce597f0b83a793ee52`
- FBInk commit: `ede0a8d0a107768609fde9a68aab5f5c583be398`
- Kindle SDK commit: `b4a6c99d718a7cf74935f36105c62491b4336a61`
- KindleHF toolchain: [KindleModding/koxtoolchain 2026.04](https://github.com/KindleModding/koxtoolchain/releases/tag/2026.04), `kindlehf.tar.gz`
- Toolchain SHA-256: `d08d3a5cbbaa184cc0cc2279df1d64a71656ab2c1e0223cae324b8d30c8d73e0`

## Source archive

Download [OKIGAMI-0.6.0-third-party-sources.zip](https://github.com/UroCo-lab/OKIGAMI/releases/download/v0.6.0/OKIGAMI-0.6.0-third-party-sources.zip) from the same Release as the application. It contains:

- The full Kindle Custom Screensaver project source and its upstream workflow/build scripts.
- The full pinned FBInk source, populated with the submodules used by its Kindle `strip` build:
  - `font8x8`: `fa74831b9076805271cdedc0ee12137aa548e524`
  - `stb`: `0c967ae375578a3f9e5a0952ca86c6908ddff0b1`
  - `libunibreak`: `32dc4fabf20176aeaa63c4880cc53cc123e53261`
- The pinned Kindle SDK source snapshot.
- `build-metadata.txt`, license notices, and archive provenance hashes.

FBInk's i2c-tools submodule is for Kobo, and its libevdev submodule is used by separate proof-of-concept utilities. Neither is part of these Kindle executables; these unused submodule directories are not populated in this source archive.

## Rebuilding upstream helpers

The authoritative commands are in `kindle-custom-screensaver/.github/workflows/build-kindlehf.yml` inside the source archive. They install the toolchain above, configure the Kindle SDK, build the X11 shield with Meson, build FBInk with `KINDLE=1`, then link `screensaver_draw` to `Release/libfbink.a`.

The SDK workflow's original checkout followed its then-current branch. To reproduce these dependencies, use the exact SDK revision recorded above rather than a newer branch head. The populated FBInk source can be used directly without fetching its submodules.

```sh
TOOLCHAIN="$HOME/x-tools/arm-kindlehf-linux-gnueabihf/bin/arm-kindlehf-linux-gnueabihf"
make -C FBInk distclean
make -C FBInk strip KINDLE=1 CROSS_TC="$TOOLCHAIN" -j2
"$TOOLCHAIN-gcc" -std=c11 -O2 -Wall -Wextra -IFBInk \
  kindle-custom-screensaver/src/draw/screensaver_draw.c \
  FBInk/Release/libfbink.a -lm -o screensaver_draw
```

This release reuses the upstream executable bytes; it does not claim to have independently rebuilt the C helpers. Their source revisions and build commands are provided with the preserved upstream notices. The OKIGAMI Go program has its own [build instructions](docs/build.md).
