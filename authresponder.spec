%define version 1
%define release 2
%define name authresponder
%define debug_package %{nil}
%define _build_id_links none

Name:           %{name}
Version:        %{version}
Release:        %{release}
Summary:        Nginx auth responder
License:        Beerware
URL:            https://github.com/b3n4kh/nginx-auth-responder
Source0:        %{name}-%{version}.%{release}.tar.gz

ExclusiveArch:  %{go_arches}
Requires: systemd nginx
BuildRequires: systemd
Requires(pre): shadow-utils

%description
Nginx Auth Responder

%prep
%setup -n %{name}

%pre
/usr/bin/getent passwd %{name} > /dev/null 2>&1 || /usr/sbin/useradd -r -M -u 1901 -s /sbin/nologin %{name}
/usr/bin/getent group nginx > /dev/null 2>&1 && /usr/sbin/usermod -aG nginx %{name}

%post
%systemd_post %{name}.service

%build
mkdir -p ./_build/src/github.com/b3n4kh/
ln -s $(pwd) ./_build/src/github.com/b3n4kh/%{name}

export GOPATH=$(pwd)/_build:%{gopath}
go build -o bin/%{name} .

%install
install -d %{buildroot}%{_bindir}
install -d %{buildroot}%{_sysconfdir}/%{name}
install -d %{buildroot}%{_unitdir}
install -p -m 755 bin/%{name} %{buildroot}%{_bindir}
install -p -m 644 systemd/%{name}.service %{buildroot}%{_unitdir}
install -p -m 644 config.json %{buildroot}%{_sysconfdir}/%{name}

%files
%{_bindir}/%{name}
%{_unitdir}/%{name}.service
%config(noreplace) %{_sysconfdir}/%{name}/config.json

