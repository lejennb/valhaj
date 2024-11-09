# valhaj-testing
Valhaj testing utility.

### Workflow
* Commands are tested by inspecting their return values entirely or partially, while also running additional asserts to ensure that a command has changed the state of the database as expected.
* For example, a `getset 500 hi` command will return an old value which we can simply check, but we'll also be able to check if the key `500` has actually been changed to the new value by running an additional assert.

### Requirements
* Make sure that there's a `valhaj` instance actively running, otherwise the program will just exit.

### Usage
* Simply run `make clean build` and then execute the binary: `./build/testing`
