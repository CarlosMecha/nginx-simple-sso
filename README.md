# Simple SSO

This is an example of how to configure nginx for a simple, tiny, homemade SSO system. I'm
using a similar configuration for all my side projects hosted in AWS.

Based on [this post](https://developers.shopware.com/blog/2015/03/02/sso-with-nginx-authrequest-module/)
from Shopware.

## Components

- nginx: Distributes all the incoming requests between the auth service and other applications.
- site: An application that needs to be secured, it returns the request as an HTML page. This service
is not aware of users or credentials.
- auth: The SSO authentication service. Provides a login page, authenticate and logout methods. Also
a REST API to retrieve users.

## Requirements

- Docker
- Docker compose
- Go
- Make

## Usage

```bash
$ make build
$ docker-compose up
```

Then, go to your favorite browser and open [this](http://localhost:8080/). Nginx is configured
to redirect all traffic to `site`, but first it tries to authenticate the request with `auth`. When
no authentication is provided, it redirects the request to the login page. 

Links:
 - [/](http://localhost:8080/)
 - [/me](http://localhost:8080/me)
 - [/login](http://localhost:8080/login)
 - [/logout](http://localhost:8080/logout)
