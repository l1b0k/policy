# policy

## Name

*policy* - advanced DNS traffic management based on domain name policies.

## Description

*policy* The policy plugin offers domain-based policy rules, it works like `view` plugin.
It supports rule files in the Adblock Plus (ADB) format.
You can combine with `acl` to do filter, or combine with `forward` to do forward.

## Syntax

~~~ txt
policy RULE_FILE {
    base64
    period PERIOD
    cache_dir CACHE_DIR
}
~~~

- **RULE_FILE**: The path to the rule file. Could be a local file or a remote file ( `http://example.local/rule.txt` ).

- **base64**: If set, the rule file is base64 encoded.

- **PERIOD**: The period of time to reload the rule file. Useful for remote file. Unit: go format ("60m", "72h").

- **CACHE_DIR**: If set ,will download remote file at this place. Also use this file on start up.

## Examples

~~~ corefile
.:53 {
    errors
    ready

    policy /path/banner_list.txt
    
    acl . {
       block
    }
}

.:53 {
    errors
    ready

    policy /path/internal-domains.txt
    
    forward . tls://some-internal-upstream
}

.:53 {
    errors
    ready

    forward . tls://default-upstream
}
~~~

## notes

You can not use `policy` and `view` in the same block.
