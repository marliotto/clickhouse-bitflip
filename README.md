### ClickHouse Bit Flip
The tool to fix single bit flip error in binary data files of ClickHouse.  

The tool makes the copy of origin file with .bak extension before to recovery data.
**However, don't forget to make backup by himself**

Checked on ClickHouse 19.16.2.2.

### Usage

```bash
./clickhouse-bitflip filename
```

### For example
> Code: 40, e.displayText() = DB::Exception: Checksum doesn't match: corrupted data. Reference: 32eaca6117ab50a10b91e82b65de78e1. Actual: c44af50ae79ae7639ae96b59d8032bbc. Size of compressed block: 36031. The mismatch is caused by single bit flip in data block at byte 8732, bit 2. This is most likely due to hardware failure. If you receive broken data over network and the error does not repeat every time, this can be caused by bad RAM on network interface controller or bad controller itself or bad RAM on network switches or bad CPU on network switches (look at the logs on related network switches; note that TCP checksums don't help) or bad RAM on host (look at dmesg or kern.log for enormous amount of EDAC errors, ECC-related reports, Machine Check Exceptions, mcelog; note that ECC memory can fail if the number of errors is huge) or bad CPU on host. If you read data from disk, this can be caused by disk bit rott. This exception protects ClickHouse from data corruption due to hardware failures.: (while reading column Name): (while reading from part /var/lib/clickhouse/data/default/tbl/20200420_946605_946651_20/ from mark 0 with max_rows_to_read = 8192) (version 19.16.2.2 (official build)) (from 172.18.0.1:51644) (in query: SELECT Name FROM tbl)

Corrupted partition: /var/lib/clickhouse/data/default/tbl/20200420_946605_946651_20/ 
Corrupted column: Name

```bash
./clickhouse-bitflip /var/lib/clickhouse/data/default/tbl/20200420_946605_946651_20/Name.bin
```

To fix many files
```bash
find /var/lib/clickhouse/data/default/tbl/202004* -maxdepth 2 -type f -name '*.bin' -exec ./clickhouse-bitflip {} \; > result.log
```

Also, you can remove .bak files.

### Links
- https://github.com/ClickHouse/ClickHouse/blob/2b569cf26063bc13f6443199cff8c955f16d2edc/src/Compression/CompressedReadBufferBase.cpp#L42
- https://github.com/ClickHouse/ClickHouse/blob/ea6f90b4f2c3cc2f7d8b846c769b7f3e84907e47/src/Compression/CompressionInfo.h#L10
- https://habr.com/ru/company/oleg-bunin/blog/497334/
