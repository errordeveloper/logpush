## These notes are on pushing application logfiles directly to ElasticSearch.

ES bulk UDP transport, we provide a setup with a more controllable behaviour, by the means of lighter transport and buffering. However, due to this buffered nature, this will not serve quite well as a realtime interface.

For the purpose of realitme log-talining, there is no need for indexing, as this has to be as fast as possible. However, if the log format is designed to be non-human readable, a formatter of some sort woud be required.

As one would want to deliver real-time logs to either browser or terminal, it's probably easiest to provide HTTP interface with SSE. This means that logpush would need to serve it's buffered loglines in realtime, which is reasobanle thing to do if firewalled environments. If needed, authentiacation may be implemented via a proxy.