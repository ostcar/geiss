Geiss
=====

Geiss is a protocol server defined by the asgi specs. It can be used for a
Django Channels project as replacement for daphne.

See https://channels.readthedocs.io/en/latest/asgi.html for more information.

Actually it is written “Geiß” which means goat in some German dialects. A Geiß
is a cute little fellow, please do not eat it.


Install
-------

First you have to set your GOPATH. See
https://github.com/golang/go/wiki/GOPATH.

Then download and compile the server by calling

    $ go get github.com/ostcar/geiss


Configuration and start
-----------------------

The server can be started by running

    $ $GOPATH/bin/geiss

or

    $ export PATH=$PATH:$GOPATH/bin
    $ geiss

Call

    $ geiss --help

for a list of all options.

Geiss needs a channel backend to run. The only channel backend that is
supported right now is Redis. So you have to install and start Redis to run
Geiss.


Serving static files
--------------------

Geiss can serve static files. You should not do this in production but use
a webserver like nginx or Apache HTTP Server as proxy to Geiss and let them
serve the static files. But if you can't use a webserver before Geiss, then
you should use this feature. If you don't, your static files are still
served through the channel layer, but this is probably slower.

To configure Geiss to serve static files, collected with

    $ python manage.py collectstatic

start Geiss with the option `--static` that can be used multiple times:

    $ geiss --static /static/collected-static --static /media/path/to/media/


Full channels example
---------------------

Currently this software runs for all examples in https://github.com/andrewgodwin/channels-examples

You can test it with the follwing commands:

    $ git clone https://github.com/andrewgodwin/channels-examples
    $ cd channels-examples/multichat
    $ python3 -m venv .virtualenv
    $ source .virtualenv/bin/activate
    $ pip install --requirement requirements.txt
    $ python manage.py migrate
    $ python manage.py runworker &
    $ geiss

Then start a webserver and connect to localhost:8000.


Difference between daphne and Geiss
-----------------------------------

The main difference between daphne and Geiss is that daphne is written in
Python using Twisted and Geiss is written in Go. As far as I know, Twisted
is single threaded and therefore daphne runs only one one CPU. Geiss on the
other hand starts an many threads, as there are CPU cores. Of cause, you
can start more then one daphne, but you have to use an individual tcp port
for each daphne, which makes the setup harder to configure.


License
-------

MIT


How to kill a geiß
------------------

You should not kill a geiß. But if you realy have to, run

    $ killall geiss

But please don't!
