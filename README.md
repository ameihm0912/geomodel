geomodel
========

Overview
--------
geomodel is an extension to MozDef designed to trend authentication events
over time for users, and identify potentially malicious account usage
by comparing authentication events to an established GeoIP model for the user.

Over time, the system builds a model for a user containing known localities
that user typically authenticates from. By default, a locality is defined as
a region with a radius of 500km, but is configurable.

Authentication events that occur from an address Geo-located to a region
that is outside the established localities for the user (e.g., not within
500km of any known login region) results in a new entry for the user, and
a corresponding event notification in MozDef.

Authentication events are expired from the model after 30 days by default. This
can be configured to increase or reduce the lifetime of data in the model for
a user.

State index
-----------
geomodel uses an ES index to store state information across intervals and
runs for each user. Each known principal/user is represented by a document
in this index, and these documents are updated over time. ES is the only
backend supported for state storage, however the interfaces have been
abstracted so others can be added as required.

Plugins
-------
geomodel uses a plugin system to indicate which events should be queried from
the MozDef ES data store, and if required normalize the events. The plugins
configuration option in the configuration file indicates the directory that
contains the plugins.

Plugins are python scripts that read a JSON document on STDIN, parse the data
if required, and return a geomodel.pluginResult JSON document via STDOUT. The
JSON document that is sent on STDIN is a geomodel.pluginRequest struct, which
essentially just contains the raw JSON events queried from MozDef.

Plugins contain certain comment lines that are parsed by geomodel when the
plugin is loaded.

```python
# @@ okta
# @T _type okta
# @T category okta
```

At least one `@@` line is required, and at least one `@T` line is required.
`@@` indicates the name of the plugin generating data, and will be used in
any MozDef events as required. `@T` adds a terms query to the plugin. In the
previous example, geomodel will feed data into the plugin from MozDef that is
returned using a query where `_type` matches `okta`, and `category` matches
`okta`.

Once the plugins inform geomodel how to query MozDef, geomodel runs the queries
and pipes and returned events into the plugins according to the state interval
specified in the configuration file. The plugin results are returned to geomodel
where the system incorporates the data into the existing ES state index, and
creates any required events.

See plugins included in repo for examples.

Events and alerting
-------------------

When a new location is identified for a user in the model, an event is
generated and sent to MozDef. The following is an example summary field in
this event.

```
user@host.com NEWLOCATION Taipei, Taiwan access from 118.160.1.187 (test)
[deviation:12.5] last activity was from San Francisco, United States (10371 km away)
within hour before
```
