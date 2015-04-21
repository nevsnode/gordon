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

        $key = $this->params['redisQueueKey'] . ':' . $type;
        $value = json_encode((object)array(
            'Args' => $this->encodeArgs($args),
        ));

        $this->redis->rPush($key, $value);
        return true;
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
            return json_encode($arg);
        }
        return false;
    }
}
