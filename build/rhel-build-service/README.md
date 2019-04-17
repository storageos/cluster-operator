# rhel build service

This Dockerfile is a variation of [`/build/Dockerfile`](/build/Dockerfile) for
compatibility with the rhel build service. The build job fails when a Dockerfile
contains ARGs and won't render any default values for the ARGs.
This Dockerfile should remain in sync with `/build/Dockerfile` for the rest of
the parts.

The rhel build service pulls this Dockerfile to build operator container images.
