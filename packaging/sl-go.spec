# RPM spec for Fedora, openSUSE and RHEL.  Build with:
#
#   make dist && cp packaging/sl-go.spec ~/rpmbuild/SPECS/
#   rpmbuild -ba --define "_sourcedir $PWD/release" ~/rpmbuild/SPECS/sl-go.spec
#
# or let packaging/make-packages.sh fill in @SHA256@ and stage the tarball.
#
# Fedora already carries "sl" (the steam locomotive, /usr/bin/sl).  This
# package installs the same path, so it declares Conflicts and Obsoletes and
# Provides sl: install one or the other, and "dnf install sl-go" replaces the
# locomotive instead of failing.
%global debug_package %{nil}

Name:           sl-go
Version:        @VERSION@
Release:        1%{?dist}
Summary:        More useful ls: group files by purpose and point out interesting ones
License:        GPL-3.0-or-later
URL:            https://web.archive.org/web/2016/http://practicalthought.com/sl/
Source0:        sl-go-%{version}.tar.gz
BuildRequires:  golang
Conflicts:      sl
Obsoletes:      sl < 5
Provides:       sl = %{version}-%{release}

%description
sl groups files by purpose (headers, source code, web pages, documents,
images, audio, video, fonts, archives, Apple II disk images, ...), marks the
ones that stand out in the current directory (relatively large, recently
changed or read, empty, unreadable, world-writable, broken link, odd
permissions, part of a name series), condenses runs of numbered names, and
lays the result out in columns that fit the terminal.

This is a Go port of the Tcl script by Thomas A. Phelps (2012), whose home page
is offline.  Tcl hooks are replaced by external commands (--annotate), the
startup file by a configuration file created on first run.

%prep
%setup -q

%build
export GOFLAGS="-trimpath -buildmode=pie"
export CGO_ENABLED=0
go build -ldflags "-s -w" -o sl ./src

%install
install -D -m 0755 sl %{buildroot}%{_bindir}/sl
install -D -m 0644 docs/sl.1 %{buildroot}%{_mandir}/man1/sl.1
install -D -m 0644 completions/sl.bash %{buildroot}%{_datadir}/bash-completion/completions/sl
install -D -m 0644 completions/_sl %{buildroot}%{_datadir}/zsh/site-functions/_sl
install -D -m 0644 completions/sl.fish %{buildroot}%{_datadir}/fish/vendor_completions.d/sl.fish
install -D -m 0644 README.md %{buildroot}%{_docdir}/%{name}/README.md
install -D -m 0644 INSTALL.md %{buildroot}%{_docdir}/%{name}/INSTALL.md
install -D -m 0644 LICENSE %{buildroot}%{_docdir}/%{name}/LICENSE

%files
%{_bindir}/sl
%{_datadir}/bash-completion/completions/sl
%{_datadir}/zsh/site-functions/_sl
%{_datadir}/fish/vendor_completions.d/sl.fish
%{_mandir}/man1/sl.1*
%doc %{_docdir}/%{name}/README.md %{_docdir}/%{name}/INSTALL.md %{_docdir}/%{name}/LICENSE

%changelog
* @DATE@ goberghen & Arena.ai agents - @VERSION@-1
- Go port of the 2012 Tcl script sl; static binary, configuration on first run.
