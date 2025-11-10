# Quick Build Reference Card

## Fastest Local Development Build
```bash
make fast-image  # 🚀 ~1m 23s (66% faster than before!)
```

## Standard Builds
```bash
make image       # ⚡ ~1m 47s (still 56% faster!)
make -j4 image   # ⚡ ~1m 22s (parallel)
```

## What Changed?
- ✅ Local builds now compile only 2 roxctl platforms (not 11)
- ✅ Docker layers optimized for better caching
- ✅ BuildKit cache mounts speed up package installs
- ✅ Parallel builds enabled with -j4

## Before → After
- **Clean build:** 4m 2s → 1m 23s
- **Rebuilds:** ~3m → ~1m 25s
- **CLI builds:** 1m 33s → 26s

## Backward Compatible
All existing commands work unchanged! CI builds still compile all platforms.

See [BUILD_OPTIMIZATION_SUMMARY.md](BUILD_OPTIMIZATION_SUMMARY.md) for details.
