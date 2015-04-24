Goophry
=======

Goophry aims to be a very simple and basic task-queue.  
It is built utilizing Go, Redis and in this example implementation, PHP.  

There is no direct dependency on PHP, as Goophry just executes commands.
This allows the usage of any kind of script/application, regardless of the used programming language,
as long as it runs on the commandline.


## Getting Started

### Building

```sh
# get/update necessary libraries
go get -u github.com/fzzy/radix/redis

# build the binary
go build goophry.go
```

**To run the tests:**

```sh
# get/update necessary libraries
go get -u github.com/stretchr/testify

# run tests
go test ./goo/*
```

Now simply deploy the binary together with the configuration file `goophry.config.json` in the same directory.


### Usage

The Goophry binary accepts the following flags (all are optional):

Flag|Type|Description
----|----|-----------
v|bool|Set this flag to enable verbose/debugging output
c|string|Pass this flag with the path of the configuration file (overrides the default location)

Example:
```sh
goophry -v -c /path/to/config.json
```


## Architecture

A very simplified representation:
```
[goophry.php] => [Redis] => [goophry.go] => ./something.php
                                         => ./something.php
                                         => ./somethingElse.php
                                         => ./doThis.py
```

Goophry is built by using lists in Redis. These are named with the scheme `RedisQueueKey:TaskType`.  
The example implementation in `goophry.php` shows how to insert entries into Redis accordingly.

You may also want to have a look at the example directory on how to use it.


## Advanced

### Configuration Options

Field|Type|Description
-----|----|-----------
`RedisNetwork`|string|Setting needed to connect to Redis (as required by [radix](http://godoc.org/github.com/fzzy/radix/redis#Dial))
`RedisAddress`|string|Setting needed to connect to Redis (as required by [radix](http://godoc.org/github.com/fzzy/radix/redis#Dial))
`RedisQueueKey`|string|The first part of the list-names in Redis (Must be the same in `goophry.php`)
`Tasks`|string|An array of task objects _(see below)_
`ErrorCmd`|string|A command which is executed when a task failed _(see below)_

##### Task Objects

Field|Type|Description
-----|----|-----------
`Type`|string|This field defines the TaskType, it has to be used in `addTask()`
`Script`|string|The path to the script that will be executed (with the optionally passed arguments)
`Workers`|int|The number of concurrent instances which execute the configured script

**ErrorCmd** is a command that will be executed when a task returned an exit status other than 0, or created output.  
It will then execute the command and uses `Sprintf` to replace `%s` with the error/output.
The error-content will be escaped and quoted before, so there's no need to wrap `%s` in quotes.


### Task Arguments

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

**Important:** As it is not possible to pass things like objects to the scripts via commandline,
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
