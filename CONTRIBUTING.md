# Contributing Guide

## Development guidelines

Make sure nothing is unnecesarily exported outside of this library. This is to make sure users are not depending on something that may change in the future.

To achieve this keep symbols in the root `handoff` package unexported. Other internal only packages shall be created underneath the `internal` folder.

The `plugin` package is an exception to this rule as it needs to be instantiated by the user.
