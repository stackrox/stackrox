# Pipeline image for the cimemo full-pipeline CI leg: bundles the
# memoized binaries into an image, itself memoized (image receipt).
# Deliberately minimal — it exercises the composition (binary receipts
# feeding an image receipt), not the real stackrox image recipe.
FROM alpine:3
COPY bin-exp/ /binaries/
