package ftpTreeBuilder

import "github.com/rpoletaev/goftp"

func (b *FTPBuilder) getConnection() (*goftp.FTP, error) {
	ftp, err := goftp.Connect(b.FTPAddr)
	if err != nil {
		return nil, err
	}

	err = ftp.Login(b.FTPLogin, b.FTPPass)
	return ftp, err
}

//Пытаемся установить соединение 5 раз с интервалом в 10 секунд,
//если пооытка удачная, то возвращаем установленное соединение
// func (b *FTPBuilderConfig) ftpConnect() *goftp.FTP {
// 	c, err := connect(b.FTPAddr, b.FTPLogin, b.FTPPass)
// 	if err == nil {
// 		return c
// 	}

// 	for i := 0; i < 5 && err != nil; i++ {
// 		time.Sleep(10 * time.Second)
// 		c, err = connect(b.FTPAddr, b.FTPLogin, b.FTPPass)
// 		if err == nil {
// 			return c
// 		}
// 	}

// 	b.Fatalln("Не удается установить ftp соединение: ", err)
// 	return c
// }

// попытка установки соединения и авторизация с ftp
// func connect(address, login, pass string) (*goftp.FTP, error) {
// 	ftp, err := goftp.ConnectWithTimeout(address, 30*time.Second)
// 	if err != nil {
// 		return ftp, err
// 	}
// 	if login == "" && pass == "" {
// 		return ftp, err
// 	}

// 	if err = ftp.Login(login, pass); err != nil {
// 		panic("не удалось авторизоваться")
// 	}
// 	return ftp, err
// }

// Получаем соединение из пула
// func getConnection() *goftp.FTP {
// 	for {
// 		select {
// 		case c := <-availableConnections:
// 			// fmt.Println("return connection")
// 			return c
// 		}
// 	}
// }
