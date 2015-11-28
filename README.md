Gordon
=======

Gordon aims to be a provide a simple, reliable and lightweight task-queue.

It is built utilizing Go and Redis.

Gordon provides functionality to execute tasks in the background of your main application/api/service. By using Go-routines, a concurrent execution of tasks can easily be achieved.
As Gordon just executes commands, you can use any kind of script or application, as long as it runs on the command-line.


Getting Started
===

1. Build
---

Get the latest source and build the binary:
```sh
# clone repo
git clone https://github.com/nevsnode/gordon.git
cd gordon/

# get/update necessary libraries
go get -u github.com/mediocregopher/radix.v2
go get -u github.com/BurntSushi/toml
go get -u github.com/jpillora/backoff

# build the binary
go build
```

Then create a configuration file. You'll probably just want to copy the example file and name it `gordon.config.toml`.
Change the fields in the file accordingly and deploy it in the same directory as the generated binary.  

2. Run
---

Now you can start the Gordon application. It accepts the following flags (all are optional):

Flag|Type|Description
----|----|-----------
c|string|Path to the configuration file _(Overrides the default `./gordon.config.toml`)_
t|bool|Test configuration
V|bool|Show version
v|bool|Enable verbose/debugging output

Example:
```sh
./gordon -c /path/to/gordon.config.toml -v
```

3. Integrate
---

The last step is to integrate Gordon so that commands can be executed.

This is achieved by inserting entries into Redis-lists. Take a look at the section [Handling Tasks](#handling-tasks) for a brief explanation.


Handling Tasks
===

Creating Tasks
---

Gordon essentially works by waiting for entries that are inserted into Redis-lists.
It uses the [BLPOP](http://redis.io/commands/blpop) command, that blocks until an entry is added.
With this approach tasks are received and executed immediately, unless there are no free "workers".

The lists are named by this scheme:
```
$queue_key:$task_type
```

Assuming your *queue_key* is `myqueue`, and a task is configured with the *type* `update_something`, the list would be named this:
```
myqueue:update_something
```

By knowing the list-name, you are now able to trigger the execution of this task.
You only need to push a task-entry into this Redis-list by using [RPUSH](http://redis.io/commands/rpush).
The command in Redis would then look like this:
```
RPUSH myqueue:update_something '{"args":["1234"]}'
```

This will initiate the execution of the configured `Script` for the task `update_something` with the first parameter beeing `1234`.

Assuming your task is configured with *script* `/path/to/do_something.sh`, Gordon will execute this:
```
/path/to/do_something.sh 1234
```

**Structure of a task entry**

The values that are inserted to the Redis-lists have to be JSON-encoded strings, with this structure:
```json
{
    "args": [
        "param1",
        "param2"
    ]
}
```

They have to be an object with the property `args` that is an **array containing strings**.
When no parameters are needed, just pass an empty array.

Arguments that are contained in `args`, will be passed to the *script* in the exact same order.
The task above would therefor be executed like this:
```
/path/to/do_something.sh "param1" "param2"
```

Failed Tasks
---

Tasks returning an exit-code other than 0 or creating output are considered to be failed.
In some cases one might want to handle these tasks separately, for instance re-queuing them.

An `error_script`, if defined, can be executed to notify about failed tasks.
But in some cases it is useful to handle them programmatically (in addition to notifying, or instead).

It is therefor possible to enable saving of failed tasks to separate Redis-lists. To enable this functionality `failed_tasks_ttl` must be set and have a value greater than 0.

**Note:** The time-to-live value applies to the whole list, not just single entries!

These lists are named after this scheme:
```
$queue_key:$task_type:failed
```

For example:
```
myqueue:update_something:failed
```

The values in this list are the same as the normal task entries, but also include a string-property `error_message`, like this:
```json
{
    "args": [
        "param1",
        "param2"
    ],
    "error_message": "Some error happened!"
}
```

You may then use [LINDEX](http://redis.io/commands/lindex) or [LPOP](http://redis.io/commands/lpop) to retrieve failed tasks from Redis and handle them.


Libraries
===

* [Gordon PHP](https://github.com/nevsnode/gordon-php), Example library written in PHP

As Gordon just reads and inserts to Redis, you can also just use the commonly used libraries for your programming language.


Testing
===

```sh
go test ./...
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
