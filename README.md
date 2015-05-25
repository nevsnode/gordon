Goophry
=======

Goophry aims to be a very simple, basic and lightweight task-queue.  
It is built utilizing Go, Redis and in this example implementation, PHP.  

Goophry just executes commands, which allows the usage of any kind of script or application, as long as it runs on the command-line.


## Getting Started

#### 1) Setup

```sh
# get/update necessary libraries
go get -u github.com/fzzy/radix

# build the binary
go build goophry.go
```

Then create a configuration file. You'll probably just want to copy the example file and name it `goophry.config.json`.
Change the fields in the file accordingly and deploy it in the same directory as the generated binary.  

Take a look at the section [Configuration](#configuration) to understand the meaning of all fields.

#### 2) Run

Now just fire up Goophry.
The application accepts the following flags (all are optional):

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

#### 3) Integrate

The last step is to integrate Goophry, so that you can trigger the execution of tasks.
To archive that it is necessary to push specific entries into Redis lists (using [RPUSH](http://redis.io/commands/rpush)).

There is already an example implementation in PHP for this purpose (`goophry.php`).

You may also want to have a look at the `/example` directory and the section [Handling Tasks](#handling-tasks) on how to use it.


## Configuration

Field|Type|Description
-----|----|-----------
RedisNetwork|string|Setting needed to connect to Redis (as required by [radix](http://godoc.org/github.com/fzzy/radix/redis#Dial))
RedisAddress|string|Setting needed to connect to Redis (as required by [radix](http://godoc.org/github.com/fzzy/radix/redis#Dial))
RedisQueueKey|string|The first part of the list-names in Redis (Must be the same in `goophry.php`)
Tasks|string|An array of task objects _(See below)_
ErrorCmd|string|A command which is executed when a task failed _(See below)_
FailedTasksTTL|integer|The TTL in seconds for the lists storing failed tasks _(See below)_
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

**ErrorCmd** is a command that will be executed when a task returned an exit status other than 0, or created output.  
It will then execute the command and uses `Sprintf` to replace `%s` with the error/output.
The error-content will be escaped and quoted before, so there's no need to wrap `%s` in quotes.

**FailedTasksTTL** is the time-to-live for lists that store failed tasks (in seconds).  
When a task fails the `ErrorCmd` is executed. Additionally the affected tasks can be stored in separate lists, so they can be handled later on.
If this field is not set or 0 this functionality is disabled.


## Handling Tasks

*The following examples are based on the PHP example implementation.
But it should be possible to adapt these easily to other languages.*

### Adding Tasks

The tasks are expected in Redis-lists named after this scheme: `RedisQueueKey:TaskType`.  
So if your RedisQueueKey is set to `myqueue` and a task with the type `update_something` is configured, Goophry will execute tasks added to the list `myqueue:update_something`.

Entries to this list have to be JSON-encoded strings with a structure like this:
```json
{"Args":["123","456"]}
```

Assuming that the script configured for the task with type *update_something* is `/path/to/foobar.php`, Goophry will then execute the script like this:
```
/path/to/foobar.php "123" "456"
```

Arguments to the `addTask()`-method from the example class are passed in the same order, so you can execute the command like in the example above just like this:
```php
<?php
$goophry->addTask('update_something', '123', '456');
```


#### Passing objects or arrays as parameters
As it is not possible to pass things like objects or arrays to the scripts via command-line, they may be json- and base64-encoded before.

For example a call like this:
```php
<?php
$goophry->addTask('foobar', array('user' => 123));
```

Will then be executed like this:
```
/path/to/foobar.php "InsidXNlciI6MTIzfSI="
```

### Failed Tasks

In some cases it is handy to store failed tasks, so that they can be handled programmatically afterwards (instead of just notifying about them through the `ErrorCmd`).  
For these situations it is possible to define the `FailedTasksTTL`. This enables the storage of failed tasks in separate Redis-lists, named after this scheme: `RedisQueueKey:TaskType:failed`  
Taking the example from above, failed tasks would then be stored in a Redis-list with the name `myqueue:update_something:failed`.

Setting a time-to-live value to enable this feature is mandatory, to prevent filling the lists endlessly.

As this is rather specific to individual tasks there is no general solution within Goophry.  
However here is some small snippet on how to use the method `getFailedTask()` with the example class:
```php
<?php
while (false !== ($task = $goophry->getFailedTask('update_something'))) {
    // do something with the first argument (in our example '123') ...
    echo $task->getArg(0);

    // ... or re-queue the task
    $goophry->addTaskObj($task);
}
```


## Testing

```sh
# get/update necessary libraries
go get -u github.com/stretchr/testify

# run tests
go test ./goo/*
```


## License
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
