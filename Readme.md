# vpngui
A VPN Client GUI in native Go and Win32 API's compatible with Windows 7, 8, and higher. 
This source code does not include the network stack. You can develop your own VPN client library and integrate it into this source.

<p align="center">
   <img src="https://github.com/sina-ghaderi/vpngui/blob/master/sc-1.png" alt="screenshot"/>
</p>

### build the source
This package is complete and can be built on its own; however, it does not have the capability to connect to anything.  

Dependencies:
- Windows 10 or higher (64-bit) with Internet connection
- [MSYS2](https://www.msys2.org/) platform (MINGW64)
- [Inno Setup 6](https://jrsoftware.org/isdl.php) installed at `C:\Program Files (x86)\Inno Setup 6`

Setting up the build environment:
- Open the MINGW64 program and update the packages with `pacman -Syu`
- Install the required packages using `pacman -S base-devel mingw-w64-x86_64-make`
- Clone the source with `git clone https://github.com/Sina-Ghaderi/vpngui.git`
- Navigate to the `vpngui` folder and run the command `make` to start the build process
- After a successful build, the installer output will be located in the `bin/output` folder


### A few notes
The source code includes three executable files: snixconnect, launcher, and service. The snixconnect executable requires system or admin access to run. While the graphical interface itself does not need this access, it is usually necessary to set up the tunnel interface. To launch the snixconnect GUI for users without admin access, the launcher executable sends the user's session ID to the service through a named pipe. The service, running with system access, uses its [Token](https://learn.microsoft.com/en-us/windows/win32/secauthz/access-tokens) and system privileges to start the snixconnect GUI process in the user's session (using [CreateProcessAsUser](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-createprocessasusera))

