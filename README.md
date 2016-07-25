# rend-http

_Rend server to proxy simple requests to an HTTP proxy._

This server only supports very basic operations: get, set, and delete. There is
no support for any other operations. Responses to other operations are just a
simple error saying that it doesn't recognize the request.

This is a process that allows simple "dumb" memcached clients to talk to the
EVCache HTTP cache proxy via the memcached protocol. This sounds like a lot of
hops because it is a lot of hops. This project will allow reuse of our current
java client library and the infrastructure running the HTTP proxy.
