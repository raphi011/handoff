# Contributing Guide

## Development guidelines

Make sure nothing is unnecesarily exported outside of this library. This is to make sure users are not depending on something that may change in the future.

To achieve this keep symbols in the root `handoff` package unexported. Packages that are meant to be only used from within the library (such as storage) shall be created underneath the `internal` folder.

Handoff is designed to be deployable as one binary without any external dependencies. That means:

* no connections to any database servers
* no extra frontend server
* no external cache

It is also not designed to horizontally scale. That means only instance can run at a time. Changing this would add significant complexity and should not be necessary because testing distributed systems normally entails mostly waiting for network requests and typically does not require a lot of resources.

Always remember to:

* Write tests
* Write documentation
* Make sure the startup of the server remains fast
* Not import any dependencies you don't absolutely need
