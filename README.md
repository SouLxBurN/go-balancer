# go-balance

Load Balancer written in Golang.

## Configuration API

The load balancer has accepts confiugration request on port 4501.

### /config
`curl -X POST -H "Content-Type: application/json" --data "{\"hcFrequency\":30, \"retries\":5}" localhost:4501/config`

### /register
`curl -X POST -H "Content-Type: application/json" --data "{\"url\":\"http://localhost:3000\"}" localhost:4501/register`

### /deregister
`curl -X POST -H "Content-Type: application/json" --data "{\"url\":\"http://localhost:3000\"}" localhost:4501/deregister`
