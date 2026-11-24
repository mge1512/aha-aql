#
# spec file for package mcp-server-aha-aql
#
# pcd-spec-sha256: 9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262
#
# The source tarball contains vendor/ (go mod vendor), so the build needs
# no network access.

Name:           mcp-server-aha-aql
Version:        0.13.0
Release:        1%{?dist}
Summary:        MCP server for Aha! with the query language AQL
License:        MIT
URL:            https://github.com/mge1512/aha-aql
Source0:        %{name}-%{version}.tar.xz
BuildRequires:  golang(API) >= 1.24
BuildRequires:  pandoc
%if 0%{?suse_version}
BuildRequires:  systemd-rpm-macros
%{?systemd_ordering}
%else
BuildRequires:  systemd-rpm-macros
%{?systemd_requires}
%endif

%description
mcp-server-aha-aql is a small MCP server for the Aha! product management
service. A language model works with Aha! through one query language, AQL,
and two tools: query reads, and execute changes one record. The same two
tools can be called from the command line. The server serves MCP over
standard input and output and over HTTP.

%prep
%autosetup

%build
pandoc mcp-server-aha-aql.1.md -s -t man -o mcp-server-aha-aql.1
CGO_ENABLED=0 go build -mod=vendor -trimpath \
    -ldflags="-X main.serverVersion=%{version}" \
    -o mcp-server-aha-aql .

%install
install -D -m 0755 mcp-server-aha-aql %{buildroot}%{_bindir}/mcp-server-aha-aql
install -D -m 0644 mcp-server-aha-aql.1 %{buildroot}%{_mandir}/man1/mcp-server-aha-aql.1
install -D -m 0644 mcp-server-aha-aql.service %{buildroot}%{_unitdir}/mcp-server-aha-aql.service

%check
CGO_ENABLED=0 go test -mod=vendor -count=1 -timeout 15m ./independent_tests/...

%if 0%{?suse_version}
%pre
%service_add_pre mcp-server-aha-aql.service

%post
%service_add_post mcp-server-aha-aql.service

%preun
%service_del_preun mcp-server-aha-aql.service

%postun
%service_del_postun mcp-server-aha-aql.service
%else
%post
%systemd_post mcp-server-aha-aql.service

%preun
%systemd_preun mcp-server-aha-aql.service

%postun
%systemd_postun_with_restart mcp-server-aha-aql.service
%endif

%files
%license LICENSE
%doc README.md
%{_bindir}/mcp-server-aha-aql
%{_mandir}/man1/mcp-server-aha-aql.1*
%{_unitdir}/mcp-server-aha-aql.service

%changelog
* Tue Nov 24 2026 Matthias G. Eckermann <mge@mailbox.org> - 0.13.0-1
- Translation of the specification aha-aql 0.13.0
