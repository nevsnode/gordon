Goophry
=======

Goophry aims to be a provide a simple, reliable, basic and lightweight task-queue.

It is built utilizing Go and Redis.

Goophry provides functionality to execute tasks in the background of your main application/api/service. By using Go-routines, a concurrent execution of tasks can easily be achieved.
As Goophry just executes commands, you can use any kind of script or application, as long as it runs on the command-line.


Getting Started
===

1. Setup
---

Get the latest binary from the releases, build it yourself:
```sh
# get/update necessary libraries
go get -u github.com/fzzy/radix

# build the binary
go build goophry.go
```

Then create a configuration file. You'll probably just want to copy the example file and name it `goophry.config.json`.
Change the fields in the file accordingly and deploy it in the same directory as the generated binary.  

Take a look at the section [Configuration](#configuration) to understand the meaning of all fields.

2. Run
---

Now you can start the Goophry application. It accepts the following flags (all are optional):

Flag|Type|Description
----|----|-----------
V|bool|Set this flag to show the current Goophry version
v|bool|Set this flag to enable verbose/debugging output
c|string|Pass this flag with the path of the configuration file _(Overrides the default `goophry.config.json`)_
l|string|Pass this flag with the path of the logfile _(Overrides the setting from the configuration file)_

Example:
```sh
goophry -v -c /path/to/config.json -l /path/to/logfile.log
```

3. Integrate
---

The last step is to integrate Goophry, to initiate the execution of tasks.

This is achieved by inserting entries into Redis-lists. Take a look at the section [Handling Tasks](#handling-tasks) for a brief explanation.


Handling Tasks
===

Running Tasks
---

Goophry essentially works by waiting for entries that are inserted into Redis-lists. This is archived by using the [BLPOP](http://redis.io/commands/blpop) command, that blocks until an entry is added.
With this approach tasks will be received and executed immediately, unless there are no free "Workers".

The lists are named by this scheme:
```
RedisQueueKey:TaskType
```

Assuming you configured `"RedisQueueKey": "myqueue"`, and a task with the `"Type": "update_something"`, the list would be named this:
```
myqueue:update_something
```

By knowing the list-name, you are now able to initiate the execution of this task. You only need to push a task-entry into this Redis-list by using [RPUSH](http://redis.io/commands/rpush).
The command would then look like this:
```
RPUSH myqueue:update_something '{"Args":["1234"]}'
```

This will initiate the execution of the configured `Script` for the task `update_something` with the first parameter beeing `1234`.

Assuming your task is configured with `"Script": "/path/to/do_something.sh"`, Goophry will execute this:
```
/path/to/do_something.sh 1234
```

**Structure of a task entry**

The values that are inserted to the Redis-lists have to be JSON-encoded strings, with this structure:
```json
{"Args":["param1","param2"]}
```

They have to be an object with the property `Args` that is an **array containing strings**.
When no parameters are needed, just pass an empty array.

Arguments that are contained in `Args`, will be passed to the `Script` in the exact same order.
The task above would therefor be executed like this:
```
/path/to/do_something.sh "param1" "param2"
```

Failed Tasks
---

Tasks can fail by either returning an exit-code other than 0 or by creating output. In some cases one might want to handle these tasks, for instance re-queuing them.

An `ErrorScript`, if defined, can be executed to notify about failed tasks. But in some cases it is useful to handle them programmatically (additionally to notifying, or instead).

It is therefor possible to save failed tasks to separate Redis-lists. To enable this functionality `FailedTasksTTL` must be set to a value greater than 0.

**Note:** The TTL value applies to the whole list, not just single entries!

These lists are named after this scheme:
```
RedisQueueKey:TaskType:failed
```

Our example:
```
myqueue:update_something:failed
```

The values in this list are the same as the normal task entries, but also include a string-property `ErrorMessage`, like this:
```json
{"Args":["param1","param2"],"ErrorMessage":"Some error happened!"}
```

You may then use [LINDEX](http://redis.io/commands/lindex) or [LPOP](http://redis.io/commands/lpop) to retrieve failed tasks from the Redis-lists and handle them.


Libraries
===

* [Goophry PHP](https://github.com/nevsnode/goophry-php), Example library written in PHP

As Goophry just reads and inserts to Redis, you can also just use the commonly used libraries for your programming language.


Configuration
===

Field|Type|Description
-----|----|-----------
RedisAddress|string|Setting needed to connect to Redis (as required by [radix](http://godoc.org/github.com/fzzy/radix/redis#Dial))
RedisQueueKey|string|The first part of the list-names in Redis (Must be the same in `goophry.php`)
RedisNetwork|string|Setting needed to connect to Redis _(Optional, default is `tcp`, as required by [radix](http://godoc.org/github.com/fzzy/radix/redis#Dial)_)
Tasks|array|An array of task objects _(See below)_
ErrorScript|string|The path to a script that is executed when a task failed _(Optional, remove or set to an empty string to disable it. See below)_
FailedTasksTTL|integer|The TTL in seconds for the lists storing failed tasks _(Optional, See below)_
Logfile|string|The path to a logfile, instead of printing messages on the command-line _(Optional, remove or set to an empty string to disable using a logfile)_
StatsInterface|string|The address where the http-server serving usage statistics should listen to (like `ip:port`). _(Optional, remove or set to an empty string to disable the http-server)_
StatsPattern|string|The pattern that the http-server responds on (like `/RaNdOmStRiNg`) _(Optional, default is `/`)_
StatsTLSCertFile|string|Path to certificate, if the statistics should be served over https _(Optional, remove or set to an empty string if not needed)_
StatsTLSKeyFile|string|Path to private key, if the statistics should be served over https _(Optional, remove or set to an empty string if not needed)_

##### Task Objects

Field|Type|Description
-----|----|-----------
Type|string|This field defines the TaskType, this value should be unique in your configuration, as it is used to figure out which `Script` to execute
Script|string|The path to the script that will be executed (with the optionally passed arguments)
Workers|int|The number of concurrent instances that execute the configured script. _(Optional, `1` will be used as default value)_

**ErrorScript** is a script that will be executed when a task returned an exit status other than 0, or created output.  
The script will be called passing the error/output as the first parameter.

**FailedTasksTTL** is the time-to-live for lists that store failed tasks (in seconds).  
When a task fails the `ErrorScript` is executed. Additionally the affected tasks can be stored in separate lists, so they can be handled afterwards.
If this field is not set or 0 this functionality is disabled.


Testing
===

```sh
# get/update necessary libraries
go get -u github.com/stretchr/testify

# run tests
go test ./goo/*
```


License
===
The MIT License (MIT)

Copyright (c) 2015 Sven Weintuch

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
