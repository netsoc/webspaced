# Custom domains

Next-gen webspaces support custom domains! This means you can access your
webspace at `https://mydomain.com` (for example) as well as
`https://myusername.netsoc.ie`.

!!! note
    Since the migration of webspaces is now complete,
    `https://myusername.ng.nesoc.ie` will redirect to
    `https://myusername.netsoc.ie`

## DNS configuration

In order to point your domain at Netsoc's servers (and therefore your
webspace!), you'll need to make some DNS configuration changes. The exact
process will vary from provider to provider, but the records needed will be the
same. In this guide we'll use [Cloudflare](https://www.cloudflare.com/)
(which can be configured for almost any domain registrar).

!!! warning
    Depending on your setup, it can take up to 24 hours for changes to DNS to
    propagate.

### CNAME record

The main DNS record you'll need is a `CNAME` or `ALIAS` record. This makes sure
that any requests to your domain are routed to Netsoc's servers. The target for
the `CNAME` record should be **`ws-http.netsoc.ie`**. See below for an example:

![Cloudflare CNAME record](../assets/dns_cname.png)

!!! tip
    If you're using Cloudflare, click the cloud icon to disable proxying, it's
    not needed with webspaces.

!!! note
    Some DNS providers might not allow you to put a `CNAME` record at the root
    of your domain. In this case, create an `A` record instead with the
    following IP address: `134.226.83.100`. Other providers might
    allow you to create a `CNAME` record at the root, but won't correctly set up
    the DNS to point to our servers. You should also try an `A` record if a
    sucessfully created `CNAME` isn't working.

### TXT record

In order to verify that you own a domain, we also need you to create a `TXT`
record. This should be of the form `webspace:id:123`, where `123` is your user
ID. You can find your user ID by running `netsoc account info`. Example:

![Cloudflare TXT record](../assets/dns_txt.png)

## Adding the domain

Once you've set up the DNS records, you can use
`netsoc webspace domains add mysite.nul.ie` to add the domain (with the domain
used above). If you get a message saying that verification failed, remember that
DNS records can take a while to propagate.

Once the domain verifies and is added successfully, you should be able to visit
your webspace on the new domain!

!!! note
    You might a warning that the certificate for the domain is invalid. This is
    most likely because our servers haven't had a chance to obtain an SSL
    certificate for your domain yet. Try again after a few minutes. _Beware that
    many browsers will keep old TLS connections to open, so you might need to
    close and re-open the browser completely before retrying!_
