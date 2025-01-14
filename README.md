# Clash-tutorial-for-Olares

This repository is a tutorial that shares how to configure Clash for the Olares system.

# Installing Clash

First, we need to install Clash on the device. Since we want to enable TUN mode, we will choose clash-premium.
Use the [clash-premium-installer](https://github.com/Kr328/clash-premium-installer), which is an installer for clash-premium. This installer also requires clash-core to function, and you can use the backup repository [Kuingsmile/clash-core]((https://github.com/Kuingsmile/clash-core)) (the original author of Clash has unfortunately left ヽ( ຶ▮ ຶ)ﾉ!!!).


## Installation Points

- Make sure to choose the premium version of clash-core, premium version, premium version (important things should be repeated three times).

- Modify the script `installer.sh` in [clash-premium-installer](https://github.com/Kr328/clash-premium-installer). Change the github repo address from `Dreamacro/clash` to `Kuingsmile/clash-core`
    ```
    sed -i 's/Dreamacro\/clash/Kuingsmile\/clash-core/g' installer.sh
    ```
- Modify the script `scripts/setup-tun.sh`, add two rules to the nftable configuration.
  ```

  ...
  
      chain local-dns-redirect {
        type nat hook output priority 0; policy accept;
        
        ip protocol != { tcp, udp } accept
        
        meta cgroup $BYPASS_CGROUP_CLASSID accept
        ip daddr 127.0.0.0/8 accept
        ip daddr 10.0.0.0/8 accept
        
        udp dport 53 dnat $FORWARD_DNS_REDIRECT
        tcp dport 53 dnat $FORWARD_DNS_REDIRECT
    }

  ...

      chain forward-dns-redirect {
        type nat hook prerouting priority 0; policy accept;
        
        ip protocol != { tcp, udp } accept
        ip daddr 10.0.0.0/8 accept
        
        udp dport 53 dnat $FORWARD_DNS_REDIRECT
        tcp dport 53 dnat $FORWARD_DNS_REDIRECT
    }

  ...
  ```

- Now you can install the premium version clash
  ```
  ./installer.sh install
  ```

- The default path for the configuration file is /srv/clash/config.yaml.

- The Country.mmdb file will be automatically downloaded when Clash starts, but it may not download successfully (you know the reason; otherwise, why would you need a VPN? Sigh, what a strange dependency—starting a VPN requires having a working VPN ヽ( ຶ▮ ຶ)ﾉ!!!). In such cases, you’ll need to manually download Country.mmdb and copy it to the corresponding directory. You can find the download link in the logs after Clash starts, and the storage directory can also be found in the logs. If you can’t find it, the default paths are `~/.config/clash/Country.mmdb` or `/root/.config/clash/Country.mmdb`.
- Enabling TUN mode may require configuring the `/etc/systemd/resolved.conf` file, which should look something like this 👇
    ```
    DNS=127.0.0.1 
    FallbackDNS=114.114.114.114 
    DNSStubListener=no
    ```
