// +build windows

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"runtime"
	"syscall"
	"unsafe"

	clr "github.com/ropnop/go-clr"
)

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func checkOK(hr uintptr, caller string) {
	if hr != 0x0 {
		log.Fatalf("%s returned 0x%08x", caller, hr)
	}
}

func main() {
	metaHost, err := clr.GetMetaHost()
	must(err)
	versionString := "v4.0.30319"
	pwzVersion, _ := syscall.UTF16PtrFromString(versionString)
	var pRuntimeInfo uintptr
	hr := metaHost.GetRuntime(pwzVersion, &clr.IID_ICLRRuntimeInfo, &pRuntimeInfo)
	checkOK(hr, "metahost.GetRuntime")
	runtimeInfo := clr.NewICLRRuntimeInfo(pRuntimeInfo)

	var isLoadable bool
	hr = runtimeInfo.IsLoadable(&isLoadable)
	checkOK(hr, "runtimeInfo.IsLoadable")
	if !isLoadable {
		log.Fatal("[!] IsLoadable returned false. Bailing...")
	}

	hr = runtimeInfo.BindAsLegacyV2Runtime()
	checkOK(hr, "runtimeInfo.BindAsLegacyV2Runtime")

	var pRuntimeHost uintptr
	hr = runtimeInfo.GetInterface(&clr.CLSID_CorRuntimeHost, &clr.IID_ICorRuntimeHost, &pRuntimeHost)
	runtimeHost := clr.NewICORRuntimeHost(pRuntimeHost)
	hr = runtimeHost.Start()
	checkOK(hr, "runtimeHost.Start")
	fmt.Println("[+] Loaded CLR into this process")

	var pAppDomain uintptr
	var pIUnknown uintptr
	hr = runtimeHost.GetDefaultDomain(&pIUnknown)
	checkOK(hr, "runtimeHost.GetDefaultDomain")
	iu := clr.NewIUnknown(pIUnknown)
	hr = iu.QueryInterface(&clr.IID_AppDomain, &pAppDomain)
	checkOK(hr, "iu.QueryInterface")
	appDomain := clr.NewAppDomain(pAppDomain)
	fmt.Println("[+] Got default AppDomain")

	testEXEBytes, err := ioutil.ReadFile("./TestEXE.exe")
	must(err)
	runtime.KeepAlive(testEXEBytes)

	fmt.Printf("[+] Loaded %d bytes into memory from TestExe.exe\n", len(testEXEBytes))

	safeArray, err := clr.CreateSafeArray(testEXEBytes)
	must(err)
	runtime.KeepAlive(safeArray)
	fmt.Println("[+] Crated SafeArray from byte array")

	var pAssembly uintptr
	hr = appDomain.Load_3(uintptr(unsafe.Pointer(&safeArray)), &pAssembly)
	checkOK(hr, "appDomain.Load_3")
	assembly := clr.NewAssembly(pAssembly)
	fmt.Printf("[+] Executable loaded into memory at 0x%08x\n", pAssembly)

	var pEntryPointInfo uintptr
	hr = assembly.GetEntryPoint(&pEntryPointInfo)
	checkOK(hr, "assembly.GetEntryPoint")
	fmt.Printf("[+] Executable entrypoint found at 0x%08x. Calling...\n", pEntryPointInfo)
	fmt.Println("-------")
	methodInfo := clr.NewMethodInfo(pEntryPointInfo)

	var pRetCode uintptr
	nullVariant := clr.Variant{
		VT:  1,
		Val: uintptr(0),
	}
	hr = methodInfo.Invoke_3(
		nullVariant,
		uintptr(0),
		&pRetCode)

	fmt.Println("-------")

	checkOK(hr, "methodInfo.Invoke_3")
	fmt.Printf("[+] Executable returned code %d\n", pRetCode)

	appDomain.Release()
	runtimeHost.Release()
	runtimeInfo.Release()
	metaHost.Release()

}
