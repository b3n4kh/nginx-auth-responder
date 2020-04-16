# Nginx Auth Responder

This service answers to nginx [auth_request](http://nginx.org/en/docs/http/ngx_http_auth_request_module.html) requests.

Requests to this service must set three http Headers:
 * "REMOTE-USER"
 * "X-URI"
 * "X-Host"

If not all three of them are set an Error will be logged and 401 returend.

See [auth.conf](https://github.com/b3n4kh/nginx-auth-responder/blob/master/nginx/auth.conf) for an example nginx configuration.

"X-Host" does not have to correspond to an actual vhost, it can be an arbritary key to match the configblock <host>.

"X-URI" is evalued against the <location> in the configuration.
Where <location> has to be a [prefix](https://golang.org/pkg/strings/#HasPrefix) of "X-URI" to match.
"REMOTE_USER" has to be one of the elements in the <users> block of that location.

If "REMOTE_USER" is a user in the <admins> section he will get access to any location.
There is a configuration File that has to be in json Format with the following structure:



```json
{
  "hosts": {
    "<host>": {
      "locations": {
        "<location>": {
          "users": ["<allowed_user1>", "<allowd_user2>"]
        }
      }
    },
  },
  "admins": ["<admin_user>"]
}
```

See [config.json](https://github.com/b3n4kh/nginx-auth-responder/blob/master/auth-responder/config.json) for an example configuration.
