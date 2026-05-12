# Redis config
```
CONFIG SET notify-keyspace-events KhE
```
```
K     Keyspace events, published with __keyspace@<db>__ prefix.
E     Keyevent events, published with __keyevent@<db>__ prefix.
g     Generic commands (non-type specific) like DEL, EXPIRE, RENAME, ...
$     String commands
l     List commands
s     Set commands
h     Hash commands
z     Sorted set commands
t     Stream commands
d     Module key type events
x     Expired events (events generated every time a key expires)
e     Evicted events (events generated when a key is evicted for maxmemory)
m     Key miss events generated when a key that doesn't exist is accessed (Note: not included in the 'A' class)
n     New key events generated whenever a new key is created (Note: not included in the 'A' class)
o     Overwritten events generated every time a key is overwritten (Note: not included in the 'A' class)
c     Type-changed events generated every time a key's type changes (Note: not included in the 'A' class)
A     Alias for "g$lshztdxe", so that the "AKE" string means all the events except "m", "n", "o" and "c".
```