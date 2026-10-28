"""Headless Ghidra dump of th10.exe functions and data tables, by address.

Requires Ghidra 12.2 (ships PyGhidra), the JDK it asks for (25; JAVA_HOME or
PATH), and GHIDRA_INSTALL_DIR set. On Python 3.14 install a JPype with a
cp314 wheel first (1.7.1+), then PyGhidra without its pinned deps:

    pip install jpype1 packaging "typing_extensions~=4.16"
    pip install --no-deps "$GHIDRA_INSTALL_DIR/Ghidra/Features/PyGhidra/pypkg/dist/"pyghidra-*.whl
    python tools/ghidra/dump.py --exe ".tmp/[th10]/th10.exe" \
        --out .tmp/ecl_dump/round3 --tag m2 \
        0x44c350 0x452ff0 --data 0x4741e0:32 --callers 0x44c350 --context 6

The first run imports and auto-analyses the exe into --project (a few
minutes); later runs reuse it. Ghidra refuses project paths with a component
starting with "." (so not under .tmp/); the default is %LOCALAPPDATA%/th10-ghidra.
Output matches the earlier rounds:
func_<addr>.c (prefixed "// tag: <tag>"), _data_tables.txt, _callers.txt.
The exe is only read; nothing is written back to it.
"""

import argparse
import os
import sys


def parse_addr(s):
    return int(s, 16)  # accepts "44c350" and "0x44c350"


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--exe", required=True, help="path to th10.exe (v1.00a)")
    ap.add_argument("--out", required=True, help="output directory")
    ap.add_argument("--project", default=os.path.join(os.environ.get("LOCALAPPDATA") or os.path.expanduser("~"), "th10-ghidra"),
                    help="Ghidra project directory (no path component may start with '.')")
    ap.add_argument("--tag", default="", help="tag line written at the top of each .c file")
    ap.add_argument("--data", action="append", default=[], metavar="ADDR:COUNT",
                    help="dump COUNT dwords at ADDR as int/float/hex")
    ap.add_argument("--callers", action="append", default=[], metavar="ADDR",
                    help="list the functions referencing ADDR")
    ap.add_argument("--context", type=int, default=0, metavar="N",
                    help="with --callers, also list the N instructions before each reference "
                         "(ZUN passes arguments in EAX/ESI/EDI, which the decompiler often drops)")
    ap.add_argument("--timeout", type=int, default=120, help="decompiler timeout per function (s)")
    ap.add_argument("funcs", nargs="*", help="function addresses (hex)")
    args = ap.parse_args()

    if not os.environ.get("GHIDRA_INSTALL_DIR"):
        sys.exit("GHIDRA_INSTALL_DIR is not set")
    os.makedirs(args.out, exist_ok=True)
    os.makedirs(args.project, exist_ok=True)

    import pyghidra
    pyghidra.start()

    with pyghidra.open_program(os.path.abspath(args.exe), project_location=os.path.abspath(args.project),
                               project_name="th10", analyze=True) as api:
        from ghidra.app.decompiler import DecompInterface
        from ghidra.util.task import ConsoleTaskMonitor

        program = api.getCurrentProgram()
        fm = program.getFunctionManager()
        listing = program.getListing()
        monitor = ConsoleTaskMonitor()
        ifc = DecompInterface()
        ifc.openProgram(program)

        for s in args.funcs:
            addr = api.toAddr(parse_addr(s))
            func = fm.getFunctionAt(addr)
            if func is None:
                func = api.createFunction(addr, None)
            if func is None:
                print("no function at %s" % addr, file=sys.stderr)
                continue
            res = ifc.decompileFunction(func, args.timeout, monitor)
            if not res.decompileCompleted():
                print("%s: %s" % (addr, res.getErrorMessage()), file=sys.stderr)
                continue
            path = os.path.join(args.out, "func_%x.c" % addr.getOffset())
            with open(path, "w", encoding="utf-8", newline="\n") as f:
                if args.tag:
                    f.write("// tag: %s\n" % args.tag)
                f.write(res.getDecompiledFunction().getC())
            print("wrote", path)

        if args.data:
            path = os.path.join(args.out, "_data_tables.txt")
            with open(path, "a", encoding="utf-8", newline="\n") as f:
                for spec in args.data:
                    a, n = spec.split(":")
                    base = parse_addr(a)
                    f.write("== DAT_%08x ==\n" % base)
                    for i in range(int(n, 0)):
                        p = api.toAddr(base + 4 * i)
                        iv = api.getInt(p)
                        fv = api.getFloat(p)
                        f.write("  [%02d] int=%-12d float=%-8.6g hex=0x%08x\n" % (i, iv, fv, iv & 0xffffffff))
            print("wrote", path)

        if args.callers:
            path = os.path.join(args.out, "_callers.txt")
            with open(path, "a", encoding="utf-8", newline="\n") as f:
                for s in args.callers:
                    addr = api.toAddr(parse_addr(s))
                    f.write("== refs to %s ==\n" % addr)
                    for ref in api.getReferencesTo(addr):
                        src = ref.getFromAddress()
                        fn = fm.getFunctionContaining(src)
                        f.write("  %s  %-10s %s\n" % (src, ref.getReferenceType(), fn.getName() if fn else "?"))
                        here = listing.getInstructionAt(src)
                        if args.context <= 0 or here is None:
                            continue
                        before, ins = [], here
                        while len(before) < args.context:
                            ins = ins.getPrevious()
                            if ins is None:
                                break
                            before.append(ins)
                        for i in reversed(before):
                            f.write("      %s  %s\n" % (i.getAddress(), i))
                        f.write("   -> %s  %s\n" % (src, here))
            print("wrote", path)


if __name__ == "__main__":
    main()
