<?php

require __DIR__ . '/../goophry.php';

// very simple usage without many further adjustments
$goophry = new Goophry(array(
    'redisServer'   => '10.10.10.10',
    'redisQueueKey' => 'mytaskqueue',
));
$goophry->addTask('sometask', '123456');


// the example class also has a task-class, which is returned
// from the getFailedTask method. but you can also use it for
// regular uses like this:
$task = new GoophryTask();
$task->setType('sometask');
$task->addArg('123456');

$goophry->addTaskObj($task);


// you may also extend the class so you can simple create an
// instance without passing the parameters all the time
class Taskqueue extends Goophry
{
    public function __construct()
    {
        parent::__construct(array(
            'redisServer'   => '10.10.10.10',
            'redisQueueKey' => 'mytaskqueue',
        ));
    }
}
$queue = new Taskqueue();
$queue->addTask('sometask', '123456');
