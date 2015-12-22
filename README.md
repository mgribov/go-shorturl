# A simple URL shortening service in Go

## What is it
A simple web service to take a URL and return a hash, or take that hash and return a 301 redirect to the URL.   

Powered by Go and Redis  

## Examples:
### Store a URL: 
HTTP Get: http://127.0.0.1:8080/new?u=https%3A%2F%2Fgoogle.com%2F%3Fq%3Dgolang   
Returns: {"hash": "SomeHash"}   

### Get redirected to the URL
HTTP Get: http://127.0.0.1:8080/SomeHash   
Returns: 301 Redirect
