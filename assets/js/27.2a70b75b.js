(window.webpackJsonp=window.webpackJsonp||[]).push([[27],{309:function(e,t,r){"use strict";r.r(t);var a=r(14),o=Object(a.a)({},(function(){var e=this,t=e._self._c;return t("ContentSlotsDistributor",{attrs:{"slot-key":e.$parent.slotKey}},[t("h1",{attrs:{id:"what-is-postgresql"}},[t("a",{staticClass:"header-anchor",attrs:{href:"#what-is-postgresql"}},[e._v("#")]),e._v(" What is PostgreSQL?")]),e._v(" "),t("p",[e._v("PostgreSQL is a powerful, open source object-relational database system that uses and extends the SQL language combined with many features that safely store and scale the most complicated data workloads. The origins of PostgreSQL date back to 1986 as part of the "),t("a",{attrs:{href:"https://www.postgresql.org/docs/current/history.html",target:"_blank",rel:"noopener noreferrer"}},[e._v("POSTGRES"),t("OutboundLink")],1),e._v(" project at the University of California at Berkeley and has more than 35 years of active development on the core platform.")]),e._v(" "),t("p",[e._v("PostgreSQL has earned a strong reputation for its proven architecture, reliability, data integrity, robust feature set, extensibility, and the dedication of the open source community behind the software to consistently deliver performant and innovative solutions. PostgreSQL runs on "),t("a",{attrs:{href:"https://www.postgresql.org/download/",target:"_blank",rel:"noopener noreferrer"}},[e._v("all major operating systems"),t("OutboundLink")],1),e._v(", has been "),t("a",{attrs:{href:"https://en.wikipedia.org/wiki/ACID",target:"_blank",rel:"noopener noreferrer"}},[e._v("ACID"),t("OutboundLink")],1),e._v("-compliant since 2001, and has powerful add-ons such as the popular "),t("a",{attrs:{href:"https://postgis.net/",target:"_blank",rel:"noopener noreferrer"}},[e._v("PostGIS"),t("OutboundLink")],1),e._v(" geospatial database extender. It is no surprise that PostgreSQL has become the open source relational database of choice for many people and organisations.")]),e._v(" "),t("p",[t("a",{attrs:{href:"https://www.postgresql.org/docs/current/tutorial.html",target:"_blank",rel:"noopener noreferrer"}},[e._v("Getting started"),t("OutboundLink")],1),e._v(" with using PostgreSQL has never been easier - pick a project you want to build, and let PostgreSQL safely and robustly store your data.")]),e._v(" "),t("h2",{attrs:{id:"why-use-postgresql"}},[t("a",{staticClass:"header-anchor",attrs:{href:"#why-use-postgresql"}},[e._v("#")]),e._v(" Why use PostgreSQL?")]),e._v(" "),t("p",[e._v("PostgreSQL comes with "),t("a",{attrs:{href:"https://www.postgresql.org/about/featurematrix/",target:"_blank",rel:"noopener noreferrer"}},[e._v("many features"),t("OutboundLink")],1),e._v(" aimed to help developers build applications, administrators to protect data integrity and build fault-tolerant environments, and help you manage your data no matter how big or small the dataset. In addition to being "),t("a",{attrs:{href:"https://www.postgresql.org/about/license/",target:"_blank",rel:"noopener noreferrer"}},[e._v("free and open source"),t("OutboundLink")],1),e._v(", PostgreSQL is highly extensible. For example, you can define your own data types, build out custom functions, even write code from "),t("a",{attrs:{href:"https://www.postgresql.org/docs/current/xplang.html",target:"_blank",rel:"noopener noreferrer"}},[e._v("different programming languages"),t("OutboundLink")],1),e._v(" without recompiling your database!")]),e._v(" "),t("p",[e._v("PostgreSQL tries to conform with the "),t("a",{attrs:{href:"https://www.postgresql.org/docs/current/features.html",target:"_blank",rel:"noopener noreferrer"}},[e._v("SQL standard"),t("OutboundLink")],1),e._v(" where such conformance does not contradict traditional features or could lead to poor architectural decisions. Many of the features required by the SQL standard are supported, though sometimes with slightly differing syntax or function. Further moves towards conformance can be expected over time. As of the version 15 release in October 2022, PostgreSQL conforms to at least 170 of the 179 mandatory features for SQL:2016 Core conformance. As of this writing, no relational database meets full conformance with this standard.")]),e._v(" "),t("p",[e._v("Below is an inexhaustive list of various features found in PostgreSQL, with more being added in every "),t("a",{attrs:{href:"https://www.postgresql.org/developer/roadmap/",target:"_blank",rel:"noopener noreferrer"}},[e._v("major release"),t("OutboundLink")],1),e._v(":")]),e._v(" "),t("ul",[t("li",[e._v("Data Types\n"),t("ul",[t("li",[e._v("Primitives: Integer, Numeric, String, Boolean")]),e._v(" "),t("li",[e._v("Structured: Date/Time, Array, Range / Multirange, UUID")]),e._v(" "),t("li",[e._v("Document: JSON/JSONB, XML, Key-value (Hstore)")]),e._v(" "),t("li",[e._v("Geometry: Point, Line, Circle, Polygon")]),e._v(" "),t("li",[e._v("Customizations: Composite, Custom Types")])])]),e._v(" "),t("li",[e._v("Data Integrity\n"),t("ul",[t("li",[e._v("UNIQUE, NOT NULL")]),e._v(" "),t("li",[e._v("Primary Keys")]),e._v(" "),t("li",[e._v("Foreign Keys")]),e._v(" "),t("li",[e._v("Exclusion Constraints")]),e._v(" "),t("li",[e._v("Explicit Locks, Advisory Locks")])])]),e._v(" "),t("li",[e._v("Concurrency, Performance\n"),t("ul",[t("li",[e._v("Indexing: B-tree, Multicolumn, Expressions, Partial")]),e._v(" "),t("li",[e._v("Advanced Indexing: GiST, SP-Gist, KNN Gist, GIN, BRIN, Covering indexes, Bloom filters")]),e._v(" "),t("li",[e._v("Sophisticated query planner / optimizer, index-only scans, multicolumn statistics")]),e._v(" "),t("li",[e._v("Transactions, Nested Transactions (via savepoints)")]),e._v(" "),t("li",[e._v("Multi-Version concurrency Control (MVCC)")]),e._v(" "),t("li",[e._v("Parallelization of read queries and building B-tree indexes")]),e._v(" "),t("li",[e._v("Table partitioning")]),e._v(" "),t("li",[e._v("All transaction isolation levels defined in the SQL standard, including Serializable")]),e._v(" "),t("li",[e._v("Just-in-time (JIT) compilation of expressions")])])]),e._v(" "),t("li",[e._v("Reliability, Disaster Recovery\n"),t("ul",[t("li",[e._v("Write-ahead Logging (WAL)")]),e._v(" "),t("li",[e._v("Replication: Asynchronous, Synchronous, Logical")]),e._v(" "),t("li",[e._v("Point-in-time-recovery (PITR), active standbys")]),e._v(" "),t("li",[e._v("Tablespaces")])])]),e._v(" "),t("li",[e._v("Security\n"),t("ul",[t("li",[e._v("Authentication: GSSAPI, SSPI, LDAP, SCRAM-SHA-256, Certificate, and more")]),e._v(" "),t("li",[e._v("Robust access-control system")]),e._v(" "),t("li",[e._v("Column and row-level security")]),e._v(" "),t("li",[e._v("Multi-factor authentication with certificates and an additional method")])])]),e._v(" "),t("li",[e._v("Extensibility\n"),t("ul",[t("li",[e._v("Stored functions and procedures")]),e._v(" "),t("li",[e._v("Procedural Languages: PL/pgSQL, Perl, Python, and Tcl. There are other languages available through extensions, e.g. Java, JavaScript (V8), R, Lua, and Rust")]),e._v(" "),t("li",[e._v("SQL/JSON path expressions")]),e._v(" "),t("li",[e._v("Foreign data wrappers: connect to other databases or streams with a standard SQL interface")]),e._v(" "),t("li",[e._v("Customizable storage interface for tables")]),e._v(" "),t("li",[e._v("Many extensions that provide additional functionality, including PostGIS")])])]),e._v(" "),t("li",[e._v("Internationalisation, Text Search\n"),t("ul",[t("li",[e._v("Support for international character sets, e.g. through ICU collations")]),e._v(" "),t("li",[e._v("Case-insensitive and accent-insensitive collations")]),e._v(" "),t("li",[e._v("Full-text search")])])])]),e._v(" "),t("p",[e._v("There are many more features that you can discover in the PostgreSQL "),t("a",{attrs:{href:"https://www.postgresql.org/docs/",target:"_blank",rel:"noopener noreferrer"}},[e._v("documentation"),t("OutboundLink")],1),e._v(". Additionally, PostgreSQL is highly extensible: many features, such as indexes, have defined APIs so that you can build out with PostgreSQL to solve your challenges.")]),e._v(" "),t("p",[e._v("PostgreSQL has been proven to be highly scalable both in the sheer quantity of data it can manage and in the number of concurrent users it can accommodate. There are active PostgreSQL clusters in production environments that manage many terabytes of data, and specialized systems that manage petabytes.")])])}),[],!1,null,null,null);t.default=o.exports}}]);