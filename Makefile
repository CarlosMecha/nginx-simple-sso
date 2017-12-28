
build-auth:
	go build -o auth/auth ./auth/
	docker build --rm -t carlosmecha/nginx-auth ./auth/
run-auth: build-auth
	docker run --rm -p "8081:80" carlosmecha/nginx-auth

build-site:
	go build -o site/site ./site/
	docker build --rm -t carlosmecha/nginx-site ./site/
run-site: build-site
	docker run --rm -p "8082:80" carlosmecha/nginx-site

build-nginx:
	docker build --rm -t carlosmecha/nginx ./nginx/

build: build-auth build-site build-nginx

.PHONY: build-auth, run-auth, build-site, run-site, build-nginx, build