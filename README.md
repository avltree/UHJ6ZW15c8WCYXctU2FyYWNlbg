# GWP Recruitment task

This is a simple REST API + worker required by the recruitment task.

## Requirements

The following should be present on the system, which will be running the application:

* Docker
* docker-compose
* GNU Make

## Installation and running

After cloning the project code locally, run the following commands:

```
make init
make up
```

If you want to stop the app, run

```
make down
```

Of course you can operate on the containers manually, check the simple `Makefile` for info.

## Known issues

I don't know if it's present on other systems, but the MySQL container crashes on the first run on my system. I was
unable to determine the cause for the time being, but the temporary fix is simple.

If the API doesn't seem to work for you on the very first try, simply run:

```
make down
make up
```

The problem should disappear after the restart.
