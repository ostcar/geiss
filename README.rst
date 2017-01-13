geiss
============

geiss is a protocol server defined by the asgi specs. It can be used as for an
django channels project an replacement for daphne.

See https://channels.readthedocs.io/en/latest/asgi.html for more informations.

Actually it is written "Geiß" which means goat in some german dialects. A Geiß
is a cute little fellow, please try not to eat it.


Install
-------

First you have to set your gopath. See https://github.com/golang/go/wiki/GOPATH

Then download and compile the server by calling::

    $ go get github.com/ostcar/geiss


Configuration and start
-----------------------

The server can be started by running::

    $ $GOPATH/bin/geiss

or::

    $ export PATH=$PATH:$GOPATH/bin
    $ geiss

Call::

    $ geiss --help

for a list of all options.

Geis needs a channel backend to run. The only channel backend that is supported
right now is the redis channel layer. So you have to install and start redis to
run geis.


Serving static files
--------------------

geiss can serve static files. You should not do this in production but use a
webserver like nginx or apache as proxy to geiss and let them serve the static
files. But if you can't use a webserver before geiss, then you should use this
feaute. If you don't, your static files are still served through the channel
layer, but this is probably slower.

The configure geiss to serve static files, collected with::

    python manage.py collectstatic

start geiss with the option --static that can be used multiple times::

    geiss --static /static/:collected-static --static /media/:path/to/media/


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
    $ geiss

Then start a webserver and connect to localhost:8000


Difference between daphne and geiss
-----------------------------------

The main difference between daphne and geiss is that daphne is written in
python/twisted and geiss with golang. As far as I know, twisted is single
threaded and therefore daphne runs only one one CPU. Geiss on the other hand
starts an many threads, as there are CPU cores. Of cause, you can start more
then one daphne, but you have to use an individual tcp port for each daphne,
which makes the setup harder to configure.


How to kill a geiß
------------------

You shuld not kill a geiß. But if you realy have to on unix, you can call::

    $ killall geis

But please don't!
