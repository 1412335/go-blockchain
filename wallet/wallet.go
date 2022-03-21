package wallet

import "path/filepath"

const keystoreDirName = "keystore"
const AndrejAccount = "0xf57913DB69e172c0aD5018Fb0CEBf63308B2B8D7"
const BabayagaAccount = "0xca22E5F9C5ae099f64991AB356826C4d52554bF8"
const CaesarAccount = "0xbd75C01F9b4df2DCC34e48f01ae54652F955a42e"

func GetKeystoreDirPath(dir string) string {
	return filepath.Join(dir, keystoreDirName)
}
