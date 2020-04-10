package util

import javax.mail.Folder
import javax.mail.Message
import javax.mail.MessagingException
import javax.mail.Session
import javax.mail.Store
import javax.mail.URLName
import javax.mail.internet.InternetAddress
import javax.mail.search.FromTerm

class MailService {
    private Session session
    private Store store
    private Folder folder
    private final String host
    private final String username
    private final String password
    private final String protocol = "imaps"
    private final String file = "INBOX"

    MailService(String host, String username, String password) {
        this.host = host
        this.username = username
        this.password = password
    }

    void login() throws Exception {
        URLName url = new URLName(protocol, host, 993, file, username, password)

        if (session == null) {
            session = Session.getInstance(new Properties(), null)
        }
        store = session.getStore(url)

        Timer t = new Timer(10, 3)
        Exception exception = null
        while (t.IsValid()) {
            try {
                store.connect()
                break
            } catch (Exception e) {
                println "Connection to mail server failed... retrying."
                exception = e
            }
        }
        if (exception) {
            throw exception
        }
        folder = store.getFolder(url)
        folder.open(Folder.READ_WRITE)
    }

    void logout() throws MessagingException {
        try {
            folder.close(false)
            store.close()
            store = null
            session = null
        } catch (IllegalStateException ise) {
            println "Error on logout - already logged out: ${ise.toString()}"
        } catch (Exception e) {
            throw e
        }
    }

    Message[] getMessages() throws Exception {
        try {
            return folder.getMessages()
        } catch (Exception e) {
            println e.toString()
            throw e
        }
    }

    Message[] getMessagesFromSender(String from) throws Exception {
        try {
            return folder.search(new FromTerm(new InternetAddress(from)))
        } catch (Exception e) {
            println e.toString()
            throw e
        }
    }
}
