# Docker Commands Cheat Sheet

A practical collection of useful Docker and Docker Compose commands.

> **Tip:** Replace values in `<angle-brackets>` with your own values.

---

## 1. Docker Version & System Information

```bash
docker --version
docker version
docker info
docker system df
```

Check Docker service:

```bash
sudo systemctl status docker
sudo systemctl start docker
sudo systemctl stop docker
sudo systemctl restart docker
sudo systemctl enable docker
sudo systemctl enable --now docker
```

---

## 2. Docker Images

### List images

```bash
docker images
docker image ls
```

### List image IDs only

```bash
docker image ls -q
```

### Pull an image

```bash
docker pull <image>
docker pull <image>:<tag>
```

Examples:

```bash
docker pull ubuntu
docker pull ubuntu:22.04
docker pull nginx:latest
```

### Search Docker Hub

```bash
docker search <name>
```

### Inspect an image

```bash
docker image inspect <image>
```

### Image history

```bash
docker history <image>
```

### Tag an image

```bash
docker tag <source-image> <new-name>:<tag>
```

### Remove an image

```bash
docker rmi <image>
docker image rm <image>
```

### Force remove an image

```bash
docker rmi -f <image>
```

### Remove unused images

```bash
docker image prune
```

Remove all unused images, including images not referenced by containers:

```bash
docker image prune -a
```

---

## 3. Docker Containers

### List running containers

```bash
docker ps
```

### List ALL containers

```bash
docker ps -a
```

### Show only container IDs

```bash
docker ps -q
docker ps -aq
```

### Create and run a container

```bash
docker run <image>
```

Example:

```bash
docker run hello-world
```

### Run interactively

```bash
docker run -it ubuntu:22.04 bash
```

### Run in background

```bash
docker run -d <image>
```

### Give a container a name

```bash
docker run --name <container-name> <image>
```

Example:

```bash
docker run --name my-nginx nginx
```

### Run with a specific command

```bash
docker run <image> <command>
```

Example:

```bash
docker run ubuntu:22.04 echo "Hello Docker"
```

---

## 4. Container Lifecycle

### Start a stopped container

```bash
docker start <container>
```

### Stop a running container

```bash
docker stop <container>
```

### Restart a container

```bash
docker restart <container>
```

### Pause a container

```bash
docker pause <container>
```

### Unpause a container

```bash
docker unpause <container>
```

### Kill a container

```bash
docker kill <container>
```

### Remove a stopped container

```bash
docker rm <container>
```

### Force remove a running container

```bash
docker rm -f <container>
```

### Remove ALL containers

```bash
docker rm -f $(docker ps -aq)
```

> ⚠️ This removes both running and stopped containers.

### Remove stopped containers

```bash
docker container prune
```

---

## 5. Container Logs

### View logs

```bash
docker logs <container>
```

### Follow logs

```bash
docker logs -f <container>
```

### Show timestamps

```bash
docker logs -t <container>
```

### Show the last 100 lines

```bash
docker logs --tail 100 <container>
```

### Follow only the latest 100 lines

```bash
docker logs -f --tail 100 <container>
```

---

## 6. Execute Commands Inside Containers

### Open a shell

```bash
docker exec -it <container> bash
```

If Bash isn't available:

```bash
docker exec -it <container> sh
```

### Run a command

```bash
docker exec <container> <command>
```

Examples:

```bash
docker exec my-container ls
docker exec my-container env
docker exec my-container pwd
```

### Run as root

```bash
docker exec -it -u root <container> bash
```

---

## 7. Container Details

### Inspect a container

```bash
docker inspect <container>
```

### Show container processes

```bash
docker top <container>
```

### Show resource usage

```bash
docker stats
```

For one container:

```bash
docker stats <container>
```

### Show port mappings

```bash
docker port <container>
```

### Show container filesystem changes

```bash
docker diff <container>
```

---

## 8. Port Mapping

Map a host port to a container port:

```bash
docker run -d -p <host-port>:<container-port> <image>
```

Example:

```bash
docker run -d -p 8080:80 nginx
```

Now:

```text
Host port 8080 → Container port 80
```

Bind only to localhost:

```bash
docker run -d -p 127.0.0.1:8080:80 nginx
```

Map multiple ports:

```bash
docker run -d \
  -p 8080:80 \
  -p 8443:443 \
  nginx
```

---

## 9. Environment Variables

Pass an environment variable:

```bash
docker run -e APP_ENV=production <image>
```

Multiple variables:

```bash
docker run \
  -e APP_ENV=production \
  -e PORT=3000 \
  <image>
```

Use an environment file:

```bash
docker run --env-file .env <image>
```

---

## 10. Volumes

### List volumes

```bash
docker volume ls
```

### Create a volume

```bash
docker volume create <volume-name>
```

### Inspect a volume

```bash
docker volume inspect <volume-name>
```

### Mount a named volume

```bash
docker run -v <volume-name>:/path/in/container <image>
```

Example:

```bash
docker run -d \
  -v mydata:/var/lib/mysql \
  mysql
```

### Remove a volume

```bash
docker volume rm <volume-name>
```

### Remove unused volumes

```bash
docker volume prune
```

> ⚠️ Removing a volume can permanently delete data stored in it.

---

## 11. Bind Mounts

Mount a host directory into a container:

```bash
docker run -v /host/path:/container/path <image>
```

Example:

```bash
docker run -d \
  -v /home/ubuntu/app:/app \
  ubuntu
```

Read-only mount:

```bash
docker run -v /host/path:/container/path:ro <image>
```

Modern syntax:

```bash
docker run \
  --mount type=bind,source=/host/path,target=/container/path \
  <image>
```

---

## 12. Copy Files

### Container → Host

```bash
docker cp <container>:/path/file /host/path/
```

### Host → Container

```bash
docker cp /host/path/file <container>:/path/
```

Examples:

```bash
docker cp my-container:/app/log.txt .
docker cp ./config.json my-container:/app/
```

---

## 13. Docker Networks

### List networks

```bash
docker network ls
```

### Inspect a network

```bash
docker network inspect <network>
```

### Create a network

```bash
docker network create <network-name>
```

### Connect a container

```bash
docker network connect <network> <container>
```

### Disconnect a container

```bash
docker network disconnect <network> <container>
```

### Remove a network

```bash
docker network rm <network>
```

### Remove unused networks

```bash
docker network prune
```

---

## 14. Run Containers on a Custom Network

```bash
docker network create app-network
```

Run containers on it:

```bash
docker run -d \
  --name backend \
  --network app-network \
  <backend-image>
```

```bash
docker run -d \
  --name frontend \
  --network app-network \
  <frontend-image>
```

Containers on the same Docker network can communicate using container names.

For example:

```text
http://backend:3000
```

---

# Docker Compose

Modern Docker Compose uses:

```bash
docker compose
```

instead of the older:

```bash
docker-compose
```

---

## 15. Docker Compose Basics

### Check Compose version

```bash
docker compose version
```

### Start services

```bash
docker compose up
```

Start in background:

```bash
docker compose up -d
```

### Build and start

```bash
docker compose up -d --build
```

### Stop services

```bash
docker compose stop
```

### Stop and remove containers/networks

```bash
docker compose down
```

### Stop and remove containers, networks, and volumes

```bash
docker compose down -v
```

> ⚠️ `-v` can delete named volumes and their data.

### Restart services

```bash
docker compose restart
```

### List Compose services/containers

```bash
docker compose ps
```

### View Compose logs

```bash
docker compose logs
```

Follow logs:

```bash
docker compose logs -f
```

Specific service:

```bash
docker compose logs -f <service>
```

---

## 16. Compose Build

Build services:

```bash
docker compose build
```

Build without cache:

```bash
docker compose build --no-cache
```

Build and start:

```bash
docker compose up -d --build
```

---

## 17. Compose Execute Commands

Open a shell:

```bash
docker compose exec <service> bash
```

Or:

```bash
docker compose exec <service> sh
```

Run a command:

```bash
docker compose exec <service> <command>
```

Example:

```bash
docker compose exec backend ls
```

---

## 18. Compose Configuration

Use a specific Compose file:

```bash
docker compose -f docker-compose.yml up -d
```

Use another file:

```bash
docker compose -f docker-compose.prod.yml up -d
```

Multiple Compose files:

```bash
docker compose \
  -f docker-compose.yml \
  -f docker-compose.prod.yml \
  up -d
```

Validate/render the Compose configuration:

```bash
docker compose config
```

---

# Dockerfile

## 19. Build an Image

From the current directory:

```bash
docker build -t myapp:latest .
```

Specify a Dockerfile:

```bash
docker build -f Dockerfile.prod -t myapp:latest .
```

Build with a different tag:

```bash
docker build -t myapp:1.0 .
```

List resulting images:

```bash
docker image ls
```

---

## 20. Run Your Built Image

```bash
docker run myapp:latest
```

Background:

```bash
docker run -d --name myapp myapp:latest
```

With a port:

```bash
docker run -d \
  --name myapp \
  -p 8080:3000 \
  myapp:latest
```

With environment variables:

```bash
docker run -d \
  --name myapp \
  -e NODE_ENV=production \
  myapp:latest
```

---

# Docker Registry

## 21. Docker Hub Login

```bash
docker login
```

Logout:

```bash
docker logout
```

---

## 22. Tag & Push an Image

Tag:

```bash
docker tag myapp:latest <username>/myapp:latest
```

Push:

```bash
docker push <username>/myapp:latest
```

Pull:

```bash
docker pull <username>/myapp:latest
```

---

# Cleanup

## 23. See Docker Disk Usage

```bash
docker system df
```

More detailed:

```bash
docker system df -v
```

### Remove stopped containers

```bash
docker container prune
```

### Remove unused networks

```bash
docker network prune
```

### Remove unused images

```bash
docker image prune
```

### Remove unused volumes

```bash
docker volume prune
```

### General cleanup

```bash
docker system prune
```

### Aggressive cleanup

```bash
docker system prune -a
```

### Aggressive cleanup including volumes

```bash
docker system prune -a --volumes
```

> ⚠️ Be careful with `prune -a` and especially `--volumes`. They can remove resources/data you still need.

---

# Useful One-Liners

## 24. Show Running Containers

```bash
docker ps
```

## Show All Containers

```bash
docker ps -a
```

## Show All Images

```bash
docker image ls
```

## Show All Volumes

```bash
docker volume ls
```

## Show All Networks

```bash
docker network ls
```

## Remove All Containers

```bash
docker rm -f $(docker ps -aq)
```

## Remove All Stopped Containers

```bash
docker container prune
```

## Follow Logs

```bash
docker logs -f <container>
```

## Enter Container

```bash
docker exec -it <container> bash
```

## See Resource Usage

```bash
docker stats
```

## See Disk Usage

```bash
docker system df
```

---

# Finding Containers & Images

## 25. Filter Containers

By name:

```bash
docker ps -a --filter "name=<name>"
```

By status:

```bash
docker ps -a --filter "status=exited"
```

By ancestor/image:

```bash
docker ps -a --filter "ancestor=<image>"
```

## Filter Images

```bash
docker images --filter "reference=<image>"
```

---

# Container Restart Policies

## 26. Restart Automatically

Restart unless manually stopped:

```bash
docker run -d \
  --restart unless-stopped \
  --name myapp \
  <image>
```

Always restart:

```bash
docker run -d \
  --restart always \
  --name myapp \
  <image>
```

Only restart on failure:

```bash
docker run -d \
  --restart on-failure \
  --name myapp \
  <image>
```

---

# Health & Troubleshooting

## 27. Check Docker Service

```bash
sudo systemctl status docker
```

Restart Docker:

```bash
sudo systemctl restart docker
```

View Docker service logs:

```bash
sudo journalctl -u docker
```

Follow Docker daemon logs:

```bash
sudo journalctl -u docker -f
```

---

## 28. Check Container Health

```bash
docker inspect <container>
```

If a healthcheck exists:

```bash
docker inspect \
  --format='{{json .State.Health}}' \
  <container>
```

---

# Docker Info You Should Know

## 29. Container vs Image

```text
IMAGE
  ↓ docker run
CONTAINER
```

An **image** is the template.

A **container** is a running/stopped instance created from an image.

Example:

```bash
docker pull nginx
docker run -d --name web nginx
```

Here:

```text
nginx          = Image
web            = Container
```

Removing the container:

```bash
docker rm -f web
```

does **not** automatically remove the image.

---

# Common Workflow

## 30. Run a New Application

```bash
docker pull <image>
docker run -d \
  --name <container> \
  -p <host-port>:<container-port> \
  <image>
```

Check:

```bash
docker ps
```

Check logs:

```bash
docker logs -f <container>
```

Enter the container:

```bash
docker exec -it <container> bash
```

Stop:

```bash
docker stop <container>
```

Remove:

```bash
docker rm <container>
```

---

# Compose Project Workflow

## 31. Typical Production Workflow

Go to the project:

```bash
cd /path/to/project
```

Check configuration:

```bash
docker compose config
```

Build:

```bash
docker compose build
```

Start:

```bash
docker compose up -d
```

Check:

```bash
docker compose ps
```

View logs:

```bash
docker compose logs -f
```

Restart:

```bash
docker compose restart
```

Stop/remove:

```bash
docker compose down
```

Update images and recreate:

```bash
docker compose pull
docker compose up -d
```

Build after code/Dockerfile changes:

```bash
docker compose up -d --build
```

---

# ⚠️ Dangerous Commands

Think before running these:

```bash
docker rm -f $(docker ps -aq)
```

Removes **all containers**.

```bash
docker system prune -a
```

Removes unused Docker resources and unused images.

```bash
docker system prune -a --volumes
```

Can remove unused volumes and **permanently delete data**.

```bash
docker compose down -v
```

Removes Compose volumes and may **delete persistent application data**.

---

# Quick Cheat Sheet

| Task | Command |
|---|---|
| Docker version | `docker --version` |
| Docker info | `docker info` |
| Running containers | `docker ps` |
| All containers | `docker ps -a` |
| All images | `docker image ls` |
| Pull image | `docker pull <image>` |
| Run container | `docker run <image>` |
| Run in background | `docker run -d <image>` |
| Stop container | `docker stop <container>` |
| Start container | `docker start <container>` |
| Restart container | `docker restart <container>` |
| Remove container | `docker rm <container>` |
| Force remove | `docker rm -f <container>` |
| Remove all containers | `docker rm -f $(docker ps -aq)` |
| Container logs | `docker logs <container>` |
| Follow logs | `docker logs -f <container>` |
| Enter container | `docker exec -it <container> bash` |
| Inspect container | `docker inspect <container>` |
| Resource usage | `docker stats` |
| List volumes | `docker volume ls` |
| List networks | `docker network ls` |
| Build image | `docker build -t <name> .` |
| Remove image | `docker rmi <image>` |
| Compose start | `docker compose up -d` |
| Compose build/start | `docker compose up -d --build` |
| Compose status | `docker compose ps` |
| Compose logs | `docker compose logs -f` |
| Compose stop/remove | `docker compose down` |
| Compose shell | `docker compose exec <service> bash` |
| Docker disk usage | `docker system df` |
| Cleanup | `docker system prune` |

---

## Your Current Setup

For your Ubuntu 20.04.6 machine, the commands you've verified are:

```bash
docker --version
docker run hello-world
docker compose version
```

Your Compose command is:

```bash
docker compose
```

not:

```bash
docker-compose
```

