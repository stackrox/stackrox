package util

import javax.mail.Folder
import javax.mail.Message
import javax.mail.MessagingException
import javax.mail.Session
import javax.mail.Store
import javax.mail.URLName

class MailService {
    private Session session
    private Store store
    private Folder folder
    private final String protocol = "imaps"
    private final String file = "INBOX"

    void login(String host, String username, String password) throws Exception {
        URLName url = new URLName(protocol, host, 993, file, username, password)

        if (session == null) {
            session = Session.getInstance(new Properties(), null)
        }
        store = session.getStore(url)
        store.connect()
        folder = store.getFolder(url)
        folder.open(Folder.READ_WRITE)
    }

    void logout() throws MessagingException {
        folder.close(false)
        store.close()
        store = null
        session = null
    }

    Message[] getMessages() throws MessagingException {
        return folder.getMessages()
    }

    Message[] getMessagesFromSender(String from) {
        Message[] allMessages = []
        try {
            allMessages = getMessages()
        } catch (Exception e) {
            println e.toString()
            throw e
        }
        return allMessages.findAll { it.from*.toString().contains(from) }
    }
}
