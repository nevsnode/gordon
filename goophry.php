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

        $task = new GoophryTask();
        if (!$task->parseJson($value)) {
            return false;
        }
        $task->setType($type);

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
    protected $task;

    public function __construct()
    {
        $this->task = array(
            'Type'          => '',
            'Args'          => array(),
            'ErrorMessage'  => '',
        );
    }

    public function setType($type)
    {
        $this->task['Type'] = $type;
    }

    public function getType()
    {
        if (empty($this->task['Type'])) {
            return false;
        }
        return $this->task['Type'];
    }

    public function setArgs($args)
    {
        $this->task['Args'] = $args;
    }

    public function getArgs()
    {
        if (empty($this->task['Args'])) {
            return array();
        }
        return $this->task['Args'];
    }

    public function getArg($index = 0)
    {
        if (empty($this->task['Args']) || !isset($this->task['Args'][$index])) {
            return false;
        }
        return $this->task['Args'][$index];
    }

    public function addArg($arg)
    {
        if (empty($this->task['Args'])) {
            $this->task['Args'] = array();
        }
        $this->task['Args'][] = $arg;
    }

    public function setErrorMessage($msg)
    {
        $this->task['ErrorMessage'] = $msg;
    }

    public function getErrorMessage()
    {
        if (empty($this->task['ErrorMessage'])) {
            return '';
        }
        return $this->task['ErrorMessage'];
    }

    public function getJson()
    {
        return json_encode($this->task);
    }

    public function parseJson($string)
    {
        $task = json_decode($string, true);
        if (empty($task)) {
            return false;
        }
        $this->task = array_merge($this->task, $task);
        return true;
    }
}
