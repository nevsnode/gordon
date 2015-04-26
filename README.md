Goophry
=======

Goophry aims to be a very simple, basic and lightweight task-queue.  
It is built utilizing Go, Redis and in this example implementation, PHP.  

Goophry just executes commands, which allows the usage of any kind of script or application, as long as it runs on the commandline.


## Getting Started

#### 1) Setup

```sh
# get/update necessary libraries
go get -u github.com/fzzy/radix/redis

# build the binary
go build goophry.go
```

Then create a configuration file. You'll probably just want to copy the example file and name it `goophry.config.json`.
Change the fields in the file acordingly and deploy in the same directory as the generated binary.  

Take a look at the section [Configuration](#configuration) to understand the meaning of all fields.

#### 2) Run

Now just fire up Goophry.
The application accepts the following flags (all are optional):

Flag|Type|Description
----|----|-----------
v|bool|Set this flag to enable verbose/debugging output
c|string|Pass this flag with the path of the configuration file _(Overrides the default `goophry.config.json`)_
l|string|Pass this flag with the path of the logfile _(Overrides the setting from the configuration file)_

Example:
```sh
goophry -v -c /path/to/config.json -l /path/to/logfile.log
```

#### 3) Integrate

The last step is to integrate Goophry, so that you can trigger the execution of tasks, defined in your configuration.  
To archive that it is only necessary to insert entries into Redis lists.
For this purpose there is already an example implemention in PHP (`goophry.php`).

You may also want to have a look at the `/example` directory and the section [Task Arguments](#task-arguments) on how to use it.


## Configuration

Field|Type|Description
-----|----|-----------
RedisNetwork|string|Setting needed to connect to Redis (as required by [radix](http://godoc.org/github.com/fzzy/radix/redis#Dial))
RedisAddress|string|Setting needed to connect to Redis (as required by [radix](http://godoc.org/github.com/fzzy/radix/redis#Dial))
RedisQueueKey|string|The first part of the list-names in Redis (Must be the same in `goophry.php`)
Tasks|string|An array of task objects _(See below)_
ErrorCmd|string|A command which is executed when a task failed _(See below)_
StatsInterface|string|The adress where the http-server serving usage statistics should listen to (like `ip:port`). _(Optional, remove or set to an empty string to disable the http-server)_
Logfile|string|The path to a logfile, instead of printing messages on the commandline _(Optional, remove or set to an empty string to disable using a logfile)_

##### Task Objects

Field|Type|Description
-----|----|-----------
Type|string|This field defines the TaskType, it has to be used in `addTask()`
Script|string|The path to the script that will be executed (with the optionally passed arguments)
Workers|int|The number of concurrent instances that execute the configured script. _(Optional, `1` will be used as default value)_

**ErrorCmd** is a command that will be executed when a task returned an exit status other than 0, or created output.  
It will then execute the command and uses `Sprintf` to replace `%s` with the error/output.
The error-content will be escaped and quoted before, so there's no need to wrap `%s` in quotes.


## Testing

```sh
# get/update necessary libraries
go get -u github.com/stretchr/testify

# run tests
go test ./goo/*
```


## Task Arguments

In the PHP example, arguments are passed to `addTask()` in the same order as they
will be passed to the configured script.

That means when calling the addTask-method like this:
```php
<?php
$goophry->addTask('foobar', '123', '456');
```

Groophry will call the configured script (e.g. "foobar.php") like this:
```
/path/to/foobar.php "123" "456"
```

**Note:** As it is not possible to pass things like objects to the scripts via commandline,
they may be json-encoded before, as in the example class.

For example a call like this:
```php
<?php
$goophry->addTask('foobar', array('user' => 123));
```

Will then be executed like this:
```
/path/to/foobar.php "{\"user\":123}"
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
