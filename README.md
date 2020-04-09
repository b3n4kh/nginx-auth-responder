# Nginx Auth Responder

This service answers to nginx [auth_request](http://nginx.org/en/docs/http/ngx_http_auth_request_module.html) requests.

This service takes a request with two http Headers set "X-Original-URI" and "REMOTE_USER".

"X-Original-URI" is evalued against the <location> in the configuration.
Whereas "REMOTE_USER" against the elements in this blocks.


There is a configuration File that has to be in json Format with the following structure:

```json
{
    "<location>" ["<allowed_user1>", "<allowd_user2>"],
    "<other_location>" ["<allowed_user3>", "<allowd_user2>"],
}
```

