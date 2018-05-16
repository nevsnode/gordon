Gordon
=======

Gordon aims to be a provide a simple, reliable and lightweight task-queue.

It is built utilizing Go and Redis.

Gordon provides functionality to execute tasks in the background of your main application/api/service. By using Go-routines, a concurrent execution of tasks can easily be achieved.


## Getting Started

#### 1. Build

```sh
# get/update the code
go get -u github.com/nevsnode/gordon

# build the binary
go build github.com/nevsnode/gordon
```

Then create a configuration file. You'll probably just want to copy the example file and name it `gordon.config.toml`.
Change the fields in the file accordingly and deploy it in the same directory as the generated binary.

#### 2. Run

Now you can start the Gordon application. It accepts the following flags (all are optional):

Flag|Type|Description
----|----|-----------
conf|string|Path to the configuration file _(Overrides the default `./gordon.config.toml`)_
logfile|string|Path to a logfile _(Overrides the logfile configured in the configuration & can be an empty value, to use standard output)_
test|bool|Test configuration
verbose|bool|Enable verbose/debugging output
version|bool|Show version

Example:
```sh
./gordon -conf /path/to/gordon.config.toml -verbose
```

#### 3. Integrate

The last step is to integrate Gordon so that tasks can be executed.

This is achieved by inserting entries into Redis-lists. Take a look at the section [Handling Tasks](#handling-tasks) for a brief explanation.


## Handling Tasks

#### Creating Tasks

Gordon essentially works by checking for entries in specific Redis-lists.

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
RPUSH myqueue:update_something '{"args":["1234"],"env":{"foo":"bar"}}'
```

Assuming your task is configured with `script = /path/to/do_something.sh`, Gordon will execute it like this:
```
foo=bar /path/to/do_something.sh 1234
```

Assuming your task is configured with `url = https://api.business.com` instead, Gordon will execute an http-request like this:
```
URL: https://api.business.com/1234
POST-parameters: foo=bar
```

**Structure of a task entry**

The values that are inserted to the Redis-lists have to be JSON-encoded strings, with this structure:
```json
{
    "args": [
        "param1",
        "param2"
    ],
    "env": {
        "some_key": "some value"
    }
}
```

* `args`: list containing strings (used in the provided order)
* `env`: simple key-value-object

## Failed Tasks
Tasks returning an exit-code other than 0 or creating output are considered to be failed.
In some cases one might want to handle these tasks separately, for instance re-queuing them.

An `error_script` or `error_webhook`, if defined, can be used to notify about failed tasks.
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
    "env": {},
    "error_message": "Some error happened!"
}
```

You may then use [LINDEX](http://redis.io/commands/lindex) or [LPOP](http://redis.io/commands/lpop) to retrieve failed tasks from Redis and handle them.


## Libraries

* [Gordon PHP](https://github.com/nevsnode/gordon-php), Example library written in PHP

As Gordon just reads and inserts to Redis, you can also just use the commonly used libraries for your programming language.


## Credits

Kudos to the following libraries which are used by gordon:
* [mediocregopher/radix.v2](https://github.com/mediocregopher/radix.v2)
* [BurntSushi/toml](https://github.com/BurntSushi/toml)
* [jpillora/backoff](https://github.com/jpillora/backoff)
* [newrelic/go-agent](https://github.com/newrelic/go-agent)
