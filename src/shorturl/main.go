package main

import (
    "net/http"
    "github.com/garyburd/redigo/redis"
    "flag"
    "encoding/json"
    "crypto/sha256"
    "fmt"
    //"log"
)

const (
    letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456790"
    PARAM = "u"
    SECRET = "s"
)

var (
    redisAddress   = flag.String("redis-address", "127.0.0.1:6379", "Address to the Redis server")
    redisPrefix   = flag.String("redis-prefix", "shorturl:", "Prefix for redis storage")
    maxConnections = flag.Int("max-connections", 10, "Max connections to Redis")
    httpListen = flag.String("http-listen", ":8080", "HTTP listen string")
    secret = flag.String("secret", "secret", "Secret string to use when calling /new?s=<secret>&u=<url>")
)

type (
    Redirect struct {}
    NewMapping struct {}
)


func find(hash string, pool *redis.Pool, ch chan string) {
    c := pool.Get()
    defer c.Close()

    key := *redisPrefix + hash

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

    // @todo sort the url params to save some space
    hash := fmt.Sprintf("%x", sha256.Sum256([]byte(url)))[:6]

    // check if this url already exists
    xh := make(chan string)
    go find(hash, pool, xh)
    stored_url := <-xh


    if stored_url == "" {
        key := *redisPrefix + hash

        // @todo redis pipelining
        // @todo make sure its actually recorded
        c.Do("SET", key, url)
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

    http.Redirect(w, r, url, 301)

    defer redisPool.Close()
}


func (h *NewMapping) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    args := r.URL.Query()

    s := args[SECRET][0]
    url := args[PARAM][0]

    if s == *secret {

        redisPool := redis.NewPool(func() (redis.Conn, error) {
            c, err := redis.Dial("tcp", *redisAddress)

            if err != nil {
                return nil, err
            }

            return c, err
        }, *maxConnections)


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

    } else {
        w.Header().Add("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{ "error": "cannot create new entry, invalid secret"})
    }
}

func main() {
    flag.Parse();

    http.Handle("/", new(Redirect))
    http.Handle("/new", new(NewMapping))

    http.ListenAndServe(*httpListen, nil)
}


