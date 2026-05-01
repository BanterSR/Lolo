package main

import (
	"crypto/rand"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

var customEHex = "009cbd92ccef123be840deec0c6ed0547194c1e471d11b6f375e56038458fb18833e5bab2e1206b261495d7e2d1d9e5aa859e6d4b671a8ca5d78efede48e291a3f"

// 生成RSA密钥对，使用指定的大整数作为e
func generateRSAWithLargeE(e *big.Int) (*big.Int, *big.Int, *big.Int, error) {
	var p, q, n, phi, d *big.Int
	var err error

	maxAttempts := 10000
	startTime := time.Now()

	fmt.Printf("开始生成RSA密钥对，使用自定义e值 (长度: %d bits)...\n", e.BitLen())

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt%1000 == 0 {
			fmt.Printf("尝试次数: %d, 已用时: %v\n", attempt, time.Since(startTime))
		}

		p, err = rand.Prime(rand.Reader, 512)
		if err != nil {
			continue
		}

		q, err = rand.Prime(rand.Reader, 512)
		if err != nil {
			continue
		}

		// 2. 计算 n = p * q
		n = new(big.Int).Mul(p, q)

		// 3. 计算 φ(n) = (p-1) * (q-1)
		pMinus1 := new(big.Int).Sub(p, big.NewInt(1))
		qMinus1 := new(big.Int).Sub(q, big.NewInt(1))
		phi = new(big.Int).Mul(pMinus1, qMinus1)

		// 4. 验证e与φ(n)互质（gcd(e, φ(n)) = 1）
		gcd := new(big.Int).GCD(nil, nil, e, phi)
		if gcd.Cmp(big.NewInt(1)) != 0 {
			// 不互质，继续尝试
			continue
		}

		// 5. 计算私钥指数d，使得 e*d ≡ 1 mod φ(n)
		d = new(big.Int).ModInverse(e, phi)
		if d == nil {
			// 无法计算模逆元，继续尝试
			continue
		}

		// 6. 验证密钥对（可选但推荐）
		if !verifyRSAKeyPair(n, e, d) {
			continue
		}

		fmt.Printf("成功生成密钥对！尝试次数: %d, 总用时: %v\n",
			attempt, time.Since(startTime))
		return n, e, d, nil
	}

	return nil, nil, nil, fmt.Errorf("在 %d 次尝试后未能生成有效密钥对", maxAttempts)
}

// 验证RSA密钥对
func verifyRSAKeyPair(n, e, d *big.Int) bool {
	// 使用几个测试值验证
	testValues := []*big.Int{
		big.NewInt(2),
		big.NewInt(17),
		big.NewInt(123456),
	}

	for _, test := range testValues {
		// 确保测试值小于n
		if test.Cmp(n) >= 0 {
			// 如果测试值太大，用n-1代替
			test = new(big.Int).Sub(n, big.NewInt(1))
		}

		// 加密
		encrypted := new(big.Int).Exp(test, e, n)

		// 解密
		decrypted := new(big.Int).Exp(encrypted, d, n)

		// 验证
		if test.Cmp(decrypted) != 0 {
			return false
		}
	}

	return true
}

// 格式化为十六进制字符串（补全前导0到偶数长度）
func formatBigIntHexEven(num *big.Int) string {
	hexStr := num.Text(16)
	// 确保长度为偶数
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	return hexStr
}

// 格式化为固定长度的十六进制字符串
func formatBigIntHexFixed(num *big.Int, hexLength int) string {
	hexStr := num.Text(16)
	// 补全前导0到指定长度
	for len(hexStr) < hexLength {
		hexStr = "0" + hexStr
	}
	return hexStr
}

// 将RSA密钥对输出为PEM格式
func exportKeyPairToPEM(n, e, d *big.Int) (publicKeyPEM, privateKeyPEM string, err error) {
	// 1. 构建公钥PKCS#1格式的ASN.1 DER编码
	// 公钥结构: SEQUENCE { INTEGER n, INTEGER e }
	publicKeyASN1, err := asn1.Marshal(struct {
		N *big.Int
		E *big.Int
	}{n, e})
	if err != nil {
		return "", "", fmt.Errorf("公钥ASN.1编码失败: %v", err)
	}

	// 2. 构建私钥PKCS#1格式的ASN.1 DER编码
	// 私钥结构: SEQUENCE { version, n, e, d, p, q, dp, dq, qi }
	// 我们需要计算p, q和其他参数
	// 由于原始代码中没有p和q，这里简化为只包含n, e, d
	// 实际使用中应该从密钥生成时保存p和q
	privateKeyASN1, err := asn1.Marshal(struct {
		Version int
		N       *big.Int
		E       *big.Int
		D       *big.Int
	}{
		Version: 0,
		N:       n,
		E:       e,
		D:       d,
	})
	if err != nil {
		return "", "", fmt.Errorf("私钥ASN.1编码失败: %v", err)
	}

	// 3. 创建PEM块
	publicKeyBlock := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyASN1,
	}

	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyASN1,
	}

	// 4. 编码为PEM格式字符串
	publicKeyPEM = string(pem.EncodeToMemory(publicKeyBlock))
	privateKeyPEM = string(pem.EncodeToMemory(privateKeyBlock))

	return publicKeyPEM, privateKeyPEM, nil
}

// 从PEM格式读取RSA私钥
func importPrivateKeyFromPEM(pemData string) (*big.Int, *big.Int, *big.Int, error) {
	// 1. 解码PEM数据
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, nil, nil, fmt.Errorf("PEM解码失败")
	}

	// 2. 检查PEM类型
	if block.Type != "RSA PRIVATE KEY" {
		return nil, nil, nil, fmt.Errorf("无效的PEM类型: %s，期望 RSA PRIVATE KEY", block.Type)
	}

	// 3. 解析ASN.1 DER编码
	var keyData struct {
		Version int
		N       *big.Int
		E       *big.Int
		D       *big.Int
	}

	_, err := asn1.Unmarshal(block.Bytes, &keyData)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ASN.1解析失败: %v", err)
	}

	// 4. 验证版本
	if keyData.Version != 0 {
		return nil, nil, nil, fmt.Errorf("不支持的私钥版本: %d", keyData.Version)
	}

	return keyData.N, keyData.E, keyData.D, nil
}

// 从PEM格式读取RSA公钥
func importPublicKeyFromPEM(pemData string) (*big.Int, *big.Int, error) {
	// 1. 解码PEM数据
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, nil, fmt.Errorf("PEM解码失败")
	}

	// 2. 检查PEM类型
	if block.Type != "RSA PUBLIC KEY" {
		return nil, nil, fmt.Errorf("无效的PEM类型: %s，期望 RSA PUBLIC KEY", block.Type)
	}

	// 3. 解析ASN.1 DER编码
	var keyData struct {
		N *big.Int
		E *big.Int
	}

	_, err := asn1.Unmarshal(block.Bytes, &keyData)
	if err != nil {
		return nil, nil, fmt.Errorf("ASN.1解析失败: %v", err)
	}

	return keyData.N, keyData.E, nil
}

// 完整版：包含p, q等参数的私钥导出（需要修改生成函数以返回p和q）
func exportFullPrivateKeyToPEM(n, e, d, p, q *big.Int) (string, error) {
	// 计算其他RSA参数
	// dp = d mod (p-1)
	pMinus1 := new(big.Int).Sub(p, big.NewInt(1))
	dp := new(big.Int).Mod(d, pMinus1)

	// dq = d mod (q-1)
	qMinus1 := new(big.Int).Sub(q, big.NewInt(1))
	dq := new(big.Int).Mod(d, qMinus1)

	// qi = q^(-1) mod p
	qi := new(big.Int).ModInverse(q, p)

	// 构建PKCS#1私钥结构
	privateKeyASN1, err := asn1.Marshal(struct {
		Version int
		N       *big.Int
		E       *big.Int
		D       *big.Int
		P       *big.Int
		Q       *big.Int
		Dp      *big.Int
		Dq      *big.Int
		Qi      *big.Int
	}{
		Version: 0,
		N:       n,
		E:       e,
		D:       d,
		P:       p,
		Q:       q,
		Dp:      dp,
		Dq:      dq,
		Qi:      qi,
	})
	if err != nil {
		return "", fmt.Errorf("私钥ASN.1编码失败: %v", err)
	}

	// 创建PEM块
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyASN1,
	}

	return string(pem.EncodeToMemory(privateKeyBlock)), nil
}

func main() {
	fmt.Println("=== 使用FlyRSA的'e'值生成RSA密钥对 ===\n")

	// 1. 解析自定义的e值
	e := new(big.Int)
	e.SetString(customEHex, 16)

	fmt.Printf("自定义e值:\n")
	fmt.Printf("十六进制长度: %d 字符\n", len(customEHex))
	fmt.Printf("位长度: %d bits\n", e.BitLen())
	fmt.Printf("值: %s\n\n", customEHex)

	// 2. 生成RSA密钥对
	n, e, d, err := generateRSAWithLargeE(e)
	if err != nil {
		fmt.Printf("生成失败: %v\n", err)
		return
	}

	pu, pr, err := exportKeyPairToPEM(n, e, d)
	if err != nil {
		fmt.Printf("PEM生成失败: %v\n", err)
		return
	}
	fmt.Printf("私钥: %s\n \n 公钥: %s\n", pr, pu)

	// 3. 输出生成的密钥（正确格式化）
	fmt.Println("\n=== 生成的RSA密钥对 ===")

	// 模数n - 格式化为1024位（256个十六进制字符）
	fmt.Printf("\n模数 N (%d bits):\n", n.BitLen())
	nHex := formatBigIntHexFixed(n, 256) // 1024位 = 256个十六进制字符
	fmt.Printf("%s\n\n", nHex)

	// 公钥指数e - 使用原始格式，不补额外的前导0
	fmt.Printf("公钥指数 E (%d bits):\n", e.BitLen())
	eHex := formatBigIntHexEven(e) // 只补到偶数长度
	fmt.Printf("%s\n\n", eHex)

	// 私钥指数d - 格式化为与n相同的长度
	fmt.Printf("私钥指数 D (%d bits):\n", d.BitLen())
	dHex := formatBigIntHexFixed(d, 256) // 与n相同长度
	fmt.Printf("%s\n\n", dHex)

	// 4. 验证密钥对
	fmt.Println("=== 验证密钥对 ===")
	if verifyRSAKeyPair(n, e, d) {
		fmt.Println("✓ 密钥对验证通过")

		// 5. 生成Java代码格式（正确格式化）
		fmt.Println("\n=== 用于Java/FlyRSA的代码 ===")
		fmt.Println("// 在FlyRSA中使用的参数:")
		fmt.Printf("BigInteger n = new BigInteger(\"%s\", 16);\n", nHex)
		fmt.Printf("BigInteger e = new BigInteger(\"%s\", 16);\n", eHex)
		fmt.Printf("BigInteger d = new BigInteger(\"%s\", 16);\n\n", dHex)

		// 6. 测试加密解密
		testEncryption(n, e, d)
	} else {
		fmt.Println("✗ 密钥对验证失败")
	}
}

// 测试加密解密
func testEncryption(n, e, d *big.Int) {
	fmt.Println("=== 测试加密解密 ===")

	// 测试数据（要小于n）
	testMsg := "Hello, FlyRSA!"
	testInt := new(big.Int).SetBytes([]byte(testMsg))

	// 确保测试数据小于n
	if testInt.Cmp(n) >= 0 {
		fmt.Println("测试数据太大，需要分块处理")
		// 使用小一点的测试数据
		testInt = big.NewInt(123456789)
	}

	fmt.Printf("原始数据: %s\n", testMsg)
	fmt.Printf("转换为整数: %s\n", testInt.Text(16))

	// 加密
	encrypted := new(big.Int).Exp(testInt, e, n)
	fmt.Printf("加密后的密文: %s...\n", encrypted.Text(16)[:100])

	// 解密
	decrypted := new(big.Int).Exp(encrypted, d, n)
	fmt.Printf("解密后的整数: %s\n", decrypted.Text(16))

	// 转换回字符串
	decryptedMsg := string(decrypted.Bytes())
	fmt.Printf("解密后的字符串: %s\n", decryptedMsg)

	if testMsg == decryptedMsg || testInt.Cmp(decrypted) == 0 {
		fmt.Println("✓ 加解密测试成功")
	} else {
		fmt.Println("✗ 加解密测试失败")
	}
}
