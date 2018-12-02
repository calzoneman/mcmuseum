mcmuseum
========

Simple weekend hack.  mcmuseum is a minimal Minecraft Classic server (also
compatible with Classicube) for browsing old .dat files.  While my
[launcher](https://github.com/calzoneman/boomcraft) supports loading .dat files,
many files I have saved were from custom server implementations that supported
non-flowing liquids, and thus loading them in the vanilla game causes flooding
and crashes.  It's also a convenient way to browse levels by switching with the
`/goto` command.

## Classicube Compatibility

mcmuseum is compatible with the Classicube client, and also supports sending
heartbeats to Classicube's server to be listed on the public server list,
however, it does not verify the user's session (which doesn't particularly
matter since users cannot see or interact with one another).

## Level Format

mcmuseum loads levels from a simplified gzip format.  A Java program is provided
for converting Minecraft's .dat files (which are serialized Java objects) into
this format, to avoid having to implement a Java deserializer in another
language.

## License

0BSD.  See LICENSE.txt
