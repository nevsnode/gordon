<?php

class Goophry
{
    protected $params   = array();
    protected $redis    = null;

    public function __construct($params = array())
    {
        $defaults = array(
            'redisServer'   => '127.0.0.1',
            'redisPort'     => 6379,
            'redisQueueKey' => 'taskqueue',
            'redisTimeout'  => 2
        );
        $this->params = array_merge($defaults, $params);
    }

    public function addTask($type)
    {
        $args = func_get_args();
        array_shift($args);

        if (empty($type)) {
            return false;
        }

        if (!$this->connect()) {
            return false;
        }

        $key = sprintf('%s:%s', $this->params['redisQueueKey'], $type);
        $value = json_encode((object)array(
            'Args' => $this->encodeArgs($args),
        ));

        $this->redis->rPush($key, $value);
        return true;
    }

    public function addTaskObj($task)
    {
        if (!($task instanceof GoophryTask)) {
            return false;
        }

        $type = $task->getType();
        if (empty($type)) {
            return false;
        }

        $args = $task->getArgs();
        array_unshift($args, $type);

        return call_user_func_array(array($this, 'addTask'), $args);
    }

    public function getFailedTask($type)
    {
        if (empty($type)) {
            return false;
        }

        if (!$this->connect()) {
            return false;
        }

        $key = sprintf('%s:%s:failed', $this->params['redisQueueKey'], $type);

        $value = $this->redis->lPop($key);
        if (empty($value)) {
            return false;
        }

        $queueTask = json_decode($value, true);
        if (empty($queueTask)) {
            return false;
        }

        $task = new GoophryTask();
        $task->setType($type);
        $task->setArgs($queueTask['Args']);

        return $task;
    }

    protected function connect()
    {
        if (false === $this->redis) {
            // there was already a try to create an instance for redis, but it failed
            return false;
        }

        if (null === $this->redis) {
            // there's no instance of redis yet, so try to connect
            if (!class_exists('Redis')) {
                $this->redis = false;
                return false;
            }

            try {
                $this->redis = new Redis();
                if (!$this->redis->connect($this->params['redisServer'], $this->params['redisPort'], $this->params['redisTimeout'])) {
                    throw new RedisException('redis connect returned FALSE');
                }
            } catch (RedisException $e) {
                $this->redis = false;
                return false;
            }
        }

        return true;
    }

    protected function encodeArgs($args)
    {
        $return = array();

        foreach ($args as $v) {
            $v = $this->encodeArg($v);
            if (false !== $v) {
                $return[] = $v;
            }
        }

        return $return;
    }

    protected function encodeArg($arg)
    {
        if (is_string($arg)) {
            return $arg;
        } elseif (is_numeric($arg)) {
            return (string)$arg;
        } elseif (is_object($arg) || is_array($arg)) {
            return base64_encode(json_encode($arg));
        }
        return false;
    }
}

class GoophryTask
{
    protected $type;
    protected $args;

    public function setType($type)
    {
        $this->type = $type;
    }

    public function getType()
    {
        if (empty($this->type)) {
            return false;
        }
        return $this->type;
    }

    public function setArgs($args)
    {
        $this->args = $args;
    }

    public function getArgs()
    {
        if (empty($this->args)) {
            return array();
        }
        return $this->args;
    }

    public function getArg($index = 0)
    {
        if (empty($this->args) || !isset($this->args[$index])) {
            return false;
        }
        return $this->args[$index];
    }

    public function addArg($arg)
    {
        if (empty($this->args)) {
            $this->args = array();
        }
        $this->args[] = $arg;
    }
}
