import { useState } from 'react';

function useModal() {
    const [isModalOpen, setIsModalOpen] = useState(false);

    function openModal() {
        setIsModalOpen(true);
    }

    function closeModal() {
        setIsModalOpen(false);
    }

    return { isModalOpen, openModal, closeModal };
}

export default useModal;
