
NDKTOOLCHAIN="/apk/android-ndk-r13b/toolchains"
NDKPLATFORMS="/apk/android-ndk-r13b/platforms"

GCCBIN="$NDKTOOLCHAIN/arm-linux-androideabi-4.9/prebuilt/linux-x86_64/bin"
SYS_ROOT="$NDKPLATFORMS/android-14/arch-arm/"

CC="$GCCBIN/arm-linux-androideabi-gcc --sysroot=$SYS_ROOT" \
CCXX="$GCCBIN/arm-linux-androideabi-g++ --sysroot=$SYS_ROOT" \
CGO_CFLAGS="--sysroot=$SYS_ROOT" \
CGO_LDFLAGS="--sysroot=$SYS_ROOT" \
CGO_ENABLED=1 GOOS=android GOARCH=arm GOARM=7 go build -x -buildmode=c-shared -o ../libs/armeabi-v7a/libVPNClient.so


GCCBIN="$NDKTOOLCHAIN/x86-4.9/prebuilt/linux-x86_64/bin"
SYS_ROOT="$NDKPLATFORMS/android-14/arch-x86"

CC="$GCCBIN/i686-linux-android-gcc --sysroot=$SYS_ROOT" \
CCXX="$GCCBIN/i686-linux-android-g++ --sysroot=$SYS_ROOT" \
CGO_CFLAGS="--sysroot=$SYS_ROOT" \
CGO_LDFLAGS="--sysroot=$SYS_ROOT" \
CGO_ENABLED=1 GOOS=android GOARCH=386 go build -x -buildmode=c-shared -o ../libs/x86/libVPNClient.so

