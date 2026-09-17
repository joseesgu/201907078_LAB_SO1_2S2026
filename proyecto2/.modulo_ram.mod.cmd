savedcmd_modulo_ram.mod := printf '%s\n'   modulo_ram.o | awk '!x[$$0]++ { print("./"$$0) }' > modulo_ram.mod
