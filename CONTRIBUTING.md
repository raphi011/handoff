# Contributing Guide

## Development guidelines

Make sure nothing is unnecesarily exported outside of this library. This is to make sure users are not depending on something that may change in the future.

To achieve this keep symbols in the root `handoff` package unexported. Packages that are meant to be only used from within the library (such as storage) shall be created underneath the `internal` folder.
