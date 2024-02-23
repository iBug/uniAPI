# WireGuard

Currently only a single Service `wireguard.endpoint` is implemented. It emits the endpoint IP address for the given peer on the given WireGuard interface.

Configuration:

```yaml
interface: wg0
public-key: AAAAAA==
use-sudo: false
```
