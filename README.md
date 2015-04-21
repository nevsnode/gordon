Goophry
=======

Goophry aims to be a very simple and basic task-queue.
It is built utilizing Go, Redis and in this example implementation, PHP (but not bound to it).


## Building

```sh
# get necessary libraries
go get github.com/garyburd/redigo/redis
go get github.com/nightlyone/lockfile

# build the binary
go build goophry.go
```


## Example setup

##### goophry.config.json
```json
{
    "RedisNetwork": "tcp",
    "RedisAddress": "127.0.0.1:6379",
    "RedisQueueKey": "myqueue",
    "Tasks": [
        {
            "Type": "sometask",
            "Script": "./something.php",
            "Workers": 2
        }
    ],
    "Lockfile": "./goophry.lock",
    "ErrorCmd": "(echo 'Subject: Taskqueue Error'; echo %s) | sendmail mail@example.com"
}
```

##### addtask.php
```php
<?php
require 'goophry.php';

$goophry = new Goophry(array(
    'redisServer'   => '127.0.0.1',
    'redisPort'     => 6379,
    'redisQueueKey' => 'myqueue',
));

$goophry->addTask('sometask', '123');
```

##### something.php
```php
<?php
$arg = $argv[1]; // is '123'

/* do something with $arg */
```

## Architecture

A very simplified representation:
```
[goophry.php] => [Redis] => [goophry.go] => ./something.php
                                         => ./something.php
                                         => ./somethingElse.php
```

Goophry is built by using lists in Redis. These are named with the scheme `RedisQueueKey:TaskType`
The example implementation in `goophry.php` shows how to insert entries into Redis accordingly.

You may also want to have a look at the example below on how to use it.


## Configuration

Default configuration
```json
{
    "RedisNetwork": "tcp",
    "RedisAddress": "127.0.0.1:6379",
    "RedisQueueKey": "taskqueue",
    "Tasks": [
        {
            "Type": "something",
            "Script": "./something.php",
            "Workers": 2
        }
    ],
    "Lockfile": "./goophry.lock",
    "ErrorCmd": "(echo 'Subject: Taskqueue Error'; echo %s) | sendmail mail@example.com"
}
```

Field|Type|Description
-----|----|-----------
`RedisNetwork`|string|Setting needed to connect to Redis (as by [redigo](http://godoc.org/github.com/garyburd/redigo/redis#Dial))
`RedisAddress`|string|Setting needed to connect to Redis (as by [redigo](http://godoc.org/github.com/garyburd/redigo/redis#Dial))
`RedisQueueKey`|string|The first part of the list-names in Redis (Must be the same in `goophry.php`)
`Tasks`|string|An array of task objects _(see below)_
`Lockfile`|string|The path to the lockfile, to prevent multiple instances
`ErrorCmd`|string|A command which is executed when a task failed _(see below)_

*ErrorCmd* is a command that will be executed, when a task returned an exist status other than 0,
or created output. It will then execute the command and uses `Sprintf` to replace `%s` with the error/output.
The error-content will be escaped and quoted before, so there's no need to wrap `%s` in quotes.

##### Task Objects

Field|Type|Description
-----|----|-----------
Type|string|This field defines the TaskType, it has to be used in `addTask()`
Script|string|The path to the script that will be executed (with the optionally passed arguments)
Workers|int|The number of concurrent instances which execute the configured script


## Task Arguments

In the PHP example, arguments are passed to `addTask()` in the same order as they
will be passed to the configured script.

That means when calling the addTask-method like this:
```php
$goophry->addTask('foobar', '123', '456');
```

Groophry will call the configured script (e.g. "foobar.php") like this:
```sh
/path/to/foobar.php "123" "456"
```

*Important:* As it is not possible to pass things like objects to the scripts via commandline,
they may be json-encoded before, as in the example class.

For example a call like this:
```php
$goophry->addTask('foobar', array('user' => 123));
```

Will then be executed like this:
```sh
/path/to/foobar.php "{\"user\":123}"
```
