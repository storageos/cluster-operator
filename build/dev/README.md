# Building image during development

This dockerfile helps build an operator image while doing development quickly.
Unlike the release builds that performs all the download, generation and
compilation in a multi-stage docker build setup, this Dockerfile expects the
operator binaries to be built beforehand and only copies the binaries into the
container image. Run this dockerfile from the root of the project. The binaries
are copied from `build/_output/bin/`. Run `make dev-image` from the repo root
to automatically build the binaries and create the development container image.
Using this for the first time may take a while to build, but the subsequent
builds will reuse the built assets from the previous builds and will be much
faster.
