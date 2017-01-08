goasgiserver
============

goasgiserver is a protocol server defined by the asgi specs and therefore
an alternative to daphne.

See https://channels.readthedocs.io/en/latest/asgi.html for more informations.


Install
-------

First you have to set your gopath. See https://github.com/golang/go/wiki/GOPATH

Then download and compile the server by calling::

  $ go get github.com/ostcar/goasgiserver


Configuration and start
-----------------------

Currently there is no way to configure this software. It only uses the
redis channel backend and expects the redis server to be running on localhost on
port 6379. It opens the webserver on port 8000.

The server can be started be running::

    $ $GOPATH/bin/goasgiserver

or::

    $ export PATH=$PATH:$GOPATH/bin
    $ goasgiserver


Full channels example
---------------------

Currently this software runs for all examples in

https://github.com/andrewgodwin/channels-examples

You can test it with::

    $ git clone https://github.com/andrewgodwin/channels-examples
    $ cd channels-examples/multichat
    $ python3 -m venv .virtualenv
    $ source .virtualenv/bin/activate
    $ pip install -r requirements.txt
    $ python manage.py migrate
    $ python manage.py runworker &
    $ goasgiserver

Then start a webserver and connect to localhost:8000
