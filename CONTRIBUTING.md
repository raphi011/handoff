# Contributing Guide

## Development guidelines

Make sure nothing is unnecesarily exported outside of this library. This is to make sure users are not depending on something that may change in the future.

To achieve this keep symbols in the root `handoff` package unexported. Other internal only packages shall be created underneath the `internal` folder.

### Database

Don't forget to `defer` close the database result when querying rows as this will lead to hanging goroutines.
