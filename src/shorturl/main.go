package main

import (
    "net/http"
    //"log"
    //"strings"
    //"bufio"
    //"sync"
    //"time"
    "math/rand"
    "github.com/garyburd/redigo/redis"
    "flag"
    "encoding/json"
)

const (
    letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456790"
    PARAM = "u"
)

var (
    redisAddress   = flag.String("redis-address", "127.0.0.1:6379", "Address to the Redis server")
    maxConnections = flag.Int("max-connections", 10, "Max connections to Redis")
    httpListen = flag.String("http-listen", ":8080", "HTTP listen string")
)

type (
    Redirect struct {}
    NewMapping struct {}
)

func RandStringBytes(n int, pool *redis.Pool) string {
    // http://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang

    b := make([]byte, n)
    for i := range b {
        b[i] = letterBytes[rand.Int63() % int64(len(letterBytes))]
    }

    hash := string(b)

    // check if this hash was already used
    xh := make(chan string)
    go find(hash, pool, xh)
    existingUrl := <-xh

    if existingUrl != "" {
        //log.Println("hash " + hash + " already exists, url: " + existingUrl)
        return RandStringBytes(n, pool)
    }

    return hash
}

func find(key string, pool *redis.Pool, ch chan string) {
    c := pool.Get()
    defer c.Close()

    value, err := redis.String(c.Do("GET", key))

    if err == nil {
        ch <- value

    } else {
        ch <- ""
    }
}

func create(url string, pool *redis.Pool, ch chan string) {
    c := pool.Get()
    defer c.Close()

    // check if this url already exists
    xh := make(chan string)
    go find(url, pool, xh)
    hash := <-xh

    if hash == "" {
        hash = RandStringBytes(6, pool)

        // @todo redis pipelining
        // @todo make sure its actually recorded

        c.Do("SET", hash, url)

        // record the opposite for quick check
        c.Do("SET", url, hash)
    }

    ch <- hash
}


func (h *Redirect) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    hash := r.RequestURI[1:len(r.RequestURI)]

    redisPool := redis.NewPool(func() (redis.Conn, error) {
        c, err := redis.Dial("tcp", *redisAddress)

        if err != nil {
            return nil, err
        }

        return c, err
    }, *maxConnections)

    xh := make(chan string)
    go find(hash, redisPool, xh)
    url := <-xh

    //w.Header().Add("Content-Type", "application/json")
    //json.NewEncoder(w).Encode(map[string]interface{}{ "url": url})

    http.Redirect(w, r, url, 301)

    defer redisPool.Close()
}


func (h *NewMapping) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    redisPool := redis.NewPool(func() (redis.Conn, error) {
        c, err := redis.Dial("tcp", *redisAddress)

        if err != nil {
            return nil, err
        }

        return c, err
    }, *maxConnections)

    args := r.URL.Query()
    url := args[PARAM][0]

    xh := make(chan string)
    go create(url, redisPool, xh)
    hash := <-xh

    w.Header().Add("Content-Type", "application/json")

    if hash != "" {
        json.NewEncoder(w).Encode(map[string]interface{}{ "hash": hash})

    } else {
        json.NewEncoder(w).Encode(map[string]interface{}{ "error": "cannot create new entry for: " + url})
    }

    defer redisPool.Close()
}

func main() {
    flag.Parse();

    http.Handle("/", new(Redirect))
    http.Handle("/new", new(NewMapping))

    http.ListenAndServe(*httpListen, nil)
}


