build:
    gcc -fPIC -fno-stack-protector -c src/mypam.c
    sudo ld -x --shared -o /lib/security/mypam.so mypam.o
    rm mypam.o

test:
    g++ -o pam_test src/test.c -lpam -lpam_misc
