# go-balancer

A for-fun Load Balancer written in Golang.

## Features

- Load Balancing (It balances your load)
- Weighted Priority Queue (Based on pending requests)
- Configuable via API requests
- Passive Healthchecks
- Down Detection

## Getting Started

__Requirements__
- Golang 1.16+

__Instructions__
1. Checkout this repository.
2. Run `go build .` in the project directory.
3. Execute `./go-balance` to startup the server.
4. Using the configuration API below, register hosts, and adjust settings.
5. Profit 💰

### Configuration API

The load balancer accepts configuration requests on port 4501. You can cofigure the following with the configuration API:

- Health check frequency is seconds. (Default: 60s)
- Configure number of retries before failing a request. (Default: 5s)
- Configure the delay between retries. (Default: 1000ms)
- De/Register Hosts/Nodes to the load balancer.

#### POST `/config`
The `/config` endpoint is responsible for global settings on the load balancer.

`hcFrequency`: Health check freqency in seconds.

`retries`: Number retries before failing request.

`retryDelay`: Number of milliseconds between retries.

```json
{
    "hcFrequency": 30,
    "retries": 2,
    "retryDelay": 200
}
```
__Sample curl command__

`curl -X POST -H "Content-Type: application/json" --data "{\"hcFrequency\":30, \"retries\":5, \"retryDelay\":200}" localhost:4501/config`

#### POST `/register`
The `/register` endpoint is responsible to assigning host/nodes to the load balancer.

```json
{
    "url": "https://<HOST>:<PORT>"
}
```

__Sample curl command__

`curl -X POST -H "Content-Type: application/json" --data "{\"url\":\"http://localhost:3000\"}" localhost:4501/register`

#### POST `/deregister`
The `/deregister` endpoint is responsible to removing host/nodes from the load balancer.

```json
{
    "url": "https://<HOST>:<PORT>"
}
```

__Sample curl command__

`curl -X POST -H "Content-Type: application/json" --data "{\"url\":\"http://localhost:3000\"}" localhost:4501/deregister`
