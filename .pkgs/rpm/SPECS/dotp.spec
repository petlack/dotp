Name:           dotp
Version:        0.0.1.20240712.02
Release:        1%{?dist}
Summary:        Command-line tool for managing Time-based One-Time Passwords (TOTPs)

License:        GPL-3.0-or-later
URL:            https://github.com/petlack/dotp
Source:         %{name}-%{version}.tar.gz

BuildRequires:  go

%global debug_package %{nil}  # Disable automatic debuginfo package generation

%description
%{summary}

%prep
%autosetup -n %{name}-%{version}

%build
export GOOS=linux
go version
go build -a -ldflags="-linkmode=external" -o build/%{name} .

%install
install -Dm755 build/%{name} %{buildroot}%{_bindir}/%{name}

%check
go test ./...

%files
%{_bindir}/%{name}

%changelog
* Fri Jul 12 2024 John Doe <email@example.com> - %{version}-%{release}
- Initial RPM release
