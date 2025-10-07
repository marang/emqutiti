Name:           emqutiti
Version:        0.4.1
Release:        1
Summary:        Terminal MQTT client

License:        MIT
URL:            https://github.com/marang/emqutiti
Source0:        %{url}/archive/refs/tags/v%{version}.tar.gz

BuildRequires:  golang >= 1.21
Requires:       glibc

%description
Emqutiti is a polished MQTT client for the terminal built with Bubble Tea.

%prep
%setup -q

%build
go build -o emqutiti ./cmd/emqutiti

%install
install -Dm755 emqutiti %{buildroot}%{_bindir}/emqutiti

%files
%license LICENSE
%doc README.md
%{_bindir}/emqutiti

%changelog
* Thu Jan 02 2025 Emqutiti Maintainers <maintainers@example.com> - 0.4.1-1
- Initial package for openSUSE
